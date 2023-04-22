package main

import (
	"flag"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/plugins/registry/consul/v3"
	rateLimit "github.com/asim/go-micro/plugins/wrapper/ratelimiter/uber/v3"
	opentracing2 "github.com/asim/go-micro/plugins/wrapper/trace/opentracing/v3"
	"github.com/asim/go-micro/v3/server"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"pod/domain/repository"
	service2 "pod/domain/service"
	"pod/handler"
	"pod/proto/pod"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/registry"
	"github.com/opentracing/opentracing-go"
	"k8s.io/client-go/kubernetes"
	hystrix2 "pod/plugin/hystrix"

	"github.com/qucheng-tony/common"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
)

var (
	// 注册中心配置
	consulHost       = "192.168.0.105"
	consulPort int64 = 8500
	// 链路追踪
	tracerHost = "192.168.0.105"
	tracerPort = 6831
	// 熔断器
	hystrixPort = 9092
	// 监控端口
	prometheusPort = 9192
)

func main() {
	// 1、注册中心
	consul1 := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{
			// 如果放在docker-compose里， 也可以是服务名称
			"localhost:8500",
		}
	})

	// 2、配置中心, 存放经常变动的配置
	consulConfig, err := common.GetConsulConfig(consulHost, consulPort, "/micro/config")
	if err != nil {
		common.Error(err)
	}

	// 3、使用配置中心连mysql
	m := common.GetMysqlFromConsul(consulConfig, "mysql")
	// 4、初始化数据库
	dsn := m.User + ":" + m.Pwd + "@tcp(" + m.Host + ":" + m.Port + ")/" + m.DB + "?charset=utf8&parseTime=True&loc=Local"
	dbs, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置,
	}), &gorm.Config{})
	if err != nil {
		common.Error(err)
	}
	db, _ := dbs.DB()
	defer db.Close()

	// 5、添加链路追踪
	t, io, err := common.NewTracer("base", fmt.Sprintf("%s:%d", tracerHost, tracerPort))
	if err != nil {
		common.Error(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)

	// 6、作为客户端加熔断器
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()

	// 添加监听程序
	go func() {
		err = http.ListenAndServe(net.JoinHostPort("0.0.0.0", strconv.Itoa(hystrixPort)), hystrixStreamHandler)
		if err != nil {
			common.Error(err)
		}
	}()

	// 7、添加日志中心
	common.Info("日志统一在micro.log中")

	// 8、添加监控
	common.PrometheusBoot(prometheusPort)

	// 创建k8s连接
	// 在集群外使用
	var kubeConfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "kubeConfig file在当前系统中的地址")
	} else {
		kubeConfig = flag.String("kubeConfig", "", "kubeConfig file在当前系统中的地址")
	}
	flag.Parse()
	// 创建config示例
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		common.Fatal(err.Error())
	}

	// 在集群中使用
	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	panic(err)
	//}
	// 创建程序可操作的客户端
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		common.Fatal(err.Error())
	}

	// 创建服务实例
	service := micro.NewService(
		micro.Server(server.NewServer(func(options *server.Options) {
			options.Advertise = fmt.Sprintf("%s:8081", consulHost)
		})),
		micro.Name("go.micro.service.pod"),
		micro.Version("latest"),
		// 抛出端口
		micro.Address(":8081"),
		// 添加注册中心
		micro.Registry(consul1),
		// 添加链路追踪
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
		// 添加熔断
		micro.WrapClient(hystrix2.NewClientHystrixWrapper()),
		// 添加限流
		micro.WrapHandler(rateLimit.NewHandlerWrapper(1000)),
	)
	// 初始化服务
	service.Init()
	// 初始化数据表
	err = repository.NewPodRepository(dbs).InitTable()
	if err != nil {
		common.Fatal(err)
	}
	// 注册句柄
	podDataService := service2.NewPodDataService(repository.NewPodRepository(dbs), clientSet)
	pod.RegisterPodHandler(service.Server(), &handler.PodHandler{PodDataService: podDataService})

	// 启动服务
	if err := service.Run(); err != nil {
		common.Error(err)
	}
}

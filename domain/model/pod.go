package model

// Pod 状态 Pending 挂起 Running 运行中 Succeeded 成功 Failed 失败 Unknown 未知
type Pod struct {
	ID           int64  `gorm:"primary_key;not_null;auto_increment" json:"id"`
	PodName      string `gorm:"unique_index;not_null" json:"pod_name"`
	PodNamespace string `gorm:"pod_namespace" json:"pod_namespace"`
	// POD 所属团队
	PodTeamID int64 `json:"pod_team_id"`
	// POD 使用CPU最小值
	PodCpuMin float32 `json:"pod_cpu_min"`
	// POD 使用CPU最大值
	PodCupMax float32 `json:"pod_cup_max"`
	// 副本数量
	PodReplicas int32 `json:"pod_replicas"`
	// POD使用的内存最小值
	PodMemoryMin float32 `json:"pod_memory_min"`
	// POD使用内存最大值
	PodMemoryMax float32 `json:"pod_memory_max"`
	// POD 开放的端口
	PodPort []PodPort `gorm:"ForeignKey:PodID" json:"pod_port"`
	// PoD 使用的环境变量
	PodEnv []PodEnv `gorm:"ForeignKey:PodID" json:"pod_env"`
	// 镜像拉取策略 Always: 总是拉取pull IfNotPresent: 默认值，本地有则使用本地镜像，不拉取  Never: 只使用本地镜像，从不拉取
	PodPullPolicy string `json:"pod_pull_policy"`
	// 重启策略 Always: 当容器失效时， 由kubelet自动重启该容器 OnFailure 当容器终止运行且退出码不为0时，由kubelet重启容器
	// Never: 不论容器运行状态如何，kubelet都不会重启容器
	// 注意: kubelet 重启失效容器时间以sync-frequency乘以2n来计算， 例如1、2、4、8等，最长5min，并在重启后10min重置该时间
	// 2、pod的重启策略与控制方式有关
	// RC和DeamonSet必须设置为Always，需要保证该容器持续运行
	// Job: OnFailure或Never，确保容器执行完后不重启
	PodRestart string `json:"pod_restart"`
	// pod的发布策略
	// 重建 recreate： 停止旧版本部署新版本
	// 滚动更新 rolling-update: 一个接一个以滚动更新方式发布新版本
	// 蓝绿 blue/green：新版本与旧版本一起存在，切换流量
	// 金丝雀 canary: 将新版本面向一部分用户发布， 然后继续全量发布
	// A/B测： 以精确的方式(HTTP头、cookie、权重等)向部分用户发新版本， A/B测实际上是一种基于数据统计出业务决策的技术，
	// 在kubernetes中并不原生支持，需要额外的一些高级组建来完成设置(比如Istio、Linkerd、Tradfik或者自定义 Nginx/Haproxy等)
	// Recreate,Custom，Rolling
	PodType string `json:"pod_type"`
	// 使用的镜像名称+tag
	PodImage string `json:"pod_image"`
	// @TODO 挂盘
	// @TODO 域名设置
}

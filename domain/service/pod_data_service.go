package service

import (
	"context"
	"fmt"
	"github.com/qucheng-tony/common"
	"github.com/qucheng-tony/pod/domain/model"
	"github.com/qucheng-tony/pod/domain/repository"
	"github.com/qucheng-tony/pod/proto/pod"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strconv"
)

type IPodDataService interface {
	AddPod(*model.Pod) (int64, error)
	DeletePod(int64) error
	UpdatePod(pod *model.Pod) error
	FindPodByID(int64) (*model.Pod, error)
	FindAllPod() ([]model.Pod, error)
	CreateToK8s(*pod.PodInfo) error
	DeleteFromK8s(*model.Pod) error
	UpdateToK8s(*pod.PodInfo) error
}

func NewPodDataService(podRepository repository.IpodRepository, clientSet *kubernetes.Clientset) IPodDataService {
	return &PodDataService{
		PodRepository: podRepository,
		K8sClientSet:  clientSet,
		deployment:    &v1.Deployment{},
	}
}

type PodDataService struct {
	PodRepository repository.IpodRepository
	K8sClientSet  *kubernetes.Clientset
	deployment    *v1.Deployment
}

func (u *PodDataService) getImagePullPolicy(podInfo *pod.PodInfo) v13.PullPolicy {
	{
		switch podInfo.PodPullPolicy {
		case "Always":
			return "Always"
		case "Never":
			return "Never"
		case "IfNotPresent":
			return "IfNotPresent"
		default:
			return "Always"

		}
	}
}

func (u *PodDataService) getResources(podInfo *pod.PodInfo) (source v13.ResourceRequirements) {
	// 最大能使用多少资源
	source.Limits = v13.ResourceList{
		"cpu":    resource.MustParse(strconv.FormatFloat(float64(podInfo.PodCpuMax), 'f', 6, 64)),
		"memory": resource.MustParse(strconv.FormatFloat(float64(podInfo.PodMemoryMax), 'f', 6, 64)),
	}
	// 满足最少使用的资源量
	// @TODO 自己实现动态设置
	source.Requests = v13.ResourceList{
		"cpu":    resource.MustParse(strconv.FormatFloat(float64(podInfo.PodCpuMax), 'f', 6, 64)),
		"memory": resource.MustParse(strconv.FormatFloat(float64(podInfo.PodMemoryMax), 'f', 6, 64)),
	}
	return
}

func (u *PodDataService) getContainerPort(podInfo *pod.PodInfo) (containerPort []v13.ContainerPort) {
	for _, v := range podInfo.PodPort {
		containerPort = append(containerPort, v13.ContainerPort{
			Name:          "port-" + strconv.FormatInt(int64(v.ContainerPort), 10),
			ContainerPort: v.ContainerPort,
			Protocol:      u.getProtocol(v.Portocol),
		})
	}
	return
}

func (u *PodDataService) getProtocol(protocol string) v13.Protocol {
	switch protocol {
	case "TCP":
		return "TCP"
	case "UDP":
		return "UDP"
	case "SCTP":
		return "SCTP"
	default:
		return "TCP"
	}
}

func (u *PodDataService) getEnv(podInfo *pod.PodInfo) (envVar []v13.EnvVar) {
	for _, v := range podInfo.PodEnv {
		envVar = append(envVar, v13.EnvVar{
			Name:      v.EnvKey,
			Value:     v.EnvValue,
			ValueFrom: nil,
		})
	}
	return nil
}

// AddPod 添加pod
func (u *PodDataService) AddPod(pod2 *model.Pod) (int64, error) {
	return u.PodRepository.CreatePod(pod2)
}

// DeletePod 删除
func (u *PodDataService) DeletePod(id int64) error {
	return u.PodRepository.DeletePod(id)
}

// UpdatePod 更新
func (u *PodDataService) UpdatePod(pod *model.Pod) error {
	return u.PodRepository.UpdatePod(pod)
}

func (u *PodDataService) FindPodByID(id int64) (*model.Pod, error) {
	return u.PodRepository.FindPodByID(id)
}

func (u *PodDataService) FindAllPod() ([]model.Pod, error) {
	return u.PodRepository.FindAll()
}

func (u *PodDataService) CreateToK8s(podInfo *pod.PodInfo) error {
	u.SetDeployment(podInfo)
	if _, err := u.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Get(context.TODO(), podInfo.PodName, v12.GetOptions{}); err != nil {
		if _, err = u.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Create(context.TODO(), u.deployment, v12.CreateOptions{}); err != nil {
			common.Error(err)
			return err
		}
		common.Info("创建成功")
		return nil
	} else {
		common.Error("Pod" + podInfo.PodName + "已经存在")
		return err
	}
}

func (u *PodDataService) SetDeployment(podInfo *pod.PodInfo) {
	deployment := &v1.Deployment{}
	deployment.TypeMeta = v12.TypeMeta{
		Kind:       "deployment",
		APIVersion: "v1",
	}
	deployment.ObjectMeta = v12.ObjectMeta{
		Name:      podInfo.PodName,
		Namespace: podInfo.PodNamespace,
		Labels: map[string]string{
			"app-name": podInfo.PodName,
			"author":   "Tony",
		},
	}
	deployment.Name = podInfo.PodName
	deployment.Spec = v1.DeploymentSpec{
		// 副本个数
		Replicas: &podInfo.PodReplicas,
		Selector: &v12.LabelSelector{
			MatchLabels: map[string]string{
				"app-name": podInfo.PodName,
			},
			MatchExpressions: nil,
		},
		Template: v13.PodTemplateSpec{
			ObjectMeta: v12.ObjectMeta{
				Labels: map[string]string{
					"app-name": podInfo.PodName,
				},
			},
			Spec: v13.PodSpec{
				Containers: []v13.Container{
					{
						Name:            podInfo.PodName,
						Image:           podInfo.PodImages,
						Ports:           u.getContainerPort(podInfo),
						Env:             u.getEnv(podInfo),
						Resources:       u.getResources(podInfo),
						ImagePullPolicy: u.getImagePullPolicy(podInfo),
					},
				},
			},
		},
		Strategy:                v1.DeploymentStrategy{},
		MinReadySeconds:         0,
		RevisionHistoryLimit:    nil,
		Paused:                  false,
		ProgressDeadlineSeconds: nil,
	}
	u.deployment = deployment
}

// UpdateToK8s 更新deployment
func (u *PodDataService) UpdateToK8s(podInfo *pod.PodInfo) error {
	u.SetDeployment(podInfo)
	if _, err := u.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Get(context.TODO(), podInfo.PodName, v12.GetOptions{}); err != nil {
		common.Error(err)
		return fmt.Errorf("Pod" + podInfo.PodName + "不存在, 请先创建")
	} else {
		// 如果存在
		if _, err = u.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Update(context.TODO(), u.deployment, v12.UpdateOptions{}); err != nil {
			common.Error(err)
			return err
		}
		common.Info(podInfo.PodName + "更新成功")
	}
	return nil
}

func (u *PodDataService) DeleteFromK8s(podInfo *model.Pod) error {
	if err := u.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Delete(context.TODO(), podInfo.PodName, v12.DeleteOptions{}); err != nil {
		common.Error(err)
		return fmt.Errorf("Pod" + podInfo.PodName + "不存在, 请先创建")
	} else {
		if err := u.DeletePod(podInfo.ID); err != nil {
			common.Error(err)
			return err
		}
		common.Info("删除成功")
	}
	return nil
}

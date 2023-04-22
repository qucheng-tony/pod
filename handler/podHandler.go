package handler

import (
	"context"
	"github.com/qucheng-tony/common"
	"pod/domain/model"
	"pod/domain/service"
	"pod/proto/pod"
	"strconv"
)

type PodHandler struct {
	// 注意这里的类型是IpodDataService 类型
	PodDataService service.IPodDataService
}

func NewPodHandler(s service.PodDataService) *PodHandler {
	return &PodHandler{
		&s,
	}
}

func (e *PodHandler) AddPod(ctx context.Context, info *pod.PodInfo, rsp *pod.Response) error {
	common.Info("添加pod")
	podModel := &model.Pod{}
	err := common.SwapTo(info, podModel)
	if err != nil {
		common.Error(err)
		rsp.Msg = err.Error()
		return err
	}
	if err = e.PodDataService.CreateToK8s(info); err != nil {
		common.Error(e)
		rsp.Msg = err.Error()
		return err
	} else {
		// 操作数据库
		podID, err := e.PodDataService.AddPod(podModel)
		if err != nil {
			common.Error(e)
			rsp.Msg = err.Error()
			return err
		}
		common.Info("pod 添加成功数据库ID为:" + strconv.FormatInt(podID, 10))
		rsp.Msg = "pod 添加成功数据库ID为:" + strconv.FormatInt(podID, 10)
		return nil
	}
}

// DeletePod 删除k8s中的pod
func (e *PodHandler) DeletePod(ctx context.Context, req *pod.PodId, rsp *pod.Response) error {
	_, err := e.PodDataService.FindPodByID(req.PodId)
	if err != nil {
		common.Error(err)
		return err
	}
	if err := e.PodDataService.DeletePod(req.PodId); err != nil {
		common.Error(err)
		return err
	}
	return nil
}

// UpdatePod 更新pod
func (e *PodHandler) UpdatePod(ctx context.Context, req *pod.PodInfo, rsp *pod.Response) error {
	// 先更新k8s里的pod信息
	err := e.PodDataService.UpdateToK8s(req)
	if err != nil {
		common.Error(err)
		return err
	}
	// 查询数据库里的pod
	podModel, err := e.PodDataService.FindPodByID(req.Id)
	if err != nil {
		common.Error(err)
		return err
	}
	err = common.SwapTo(req, podModel)
	if err != nil {
		common.Error(err)
		return err
	}
	if err = e.PodDataService.UpdatePod(podModel); err != nil {
		common.Error(err)
		return err
	}
	return nil
}

func (e *PodHandler) FindPodByID(ctx context.Context, req *pod.PodId, rsp *pod.PodInfo) error {
	// 查询数据库里的数据
	podModel, err := e.PodDataService.FindPodByID(req.PodId)
	if err != nil {
		common.Error(err)
		return err
	}
	err = common.SwapTo(podModel, rsp)
	if err != nil {
		common.Error(err)
		return err
	}
	return nil
}

// FindAllPod 查询所有pod
func (e *PodHandler) FindAllPod(ctx context.Context, req *pod.FindAll, rsp *pod.AllPod) error {
	// 查询所有pod
	allPod, err := e.PodDataService.FindAllPod()
	if err != nil {
		common.Error(err)
		return err
	}
	// 整理格式
	for _, v := range allPod {
		podInfo := &pod.PodInfo{}
		err := common.SwapTo(v, podInfo)
		if err != nil {
			common.Error(err)
			return err
		}
		rsp.PodInfo = append(rsp.PodInfo, podInfo)
	}
	return nil
}

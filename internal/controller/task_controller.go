// Copyright Contributors to the KubeTask project

// Package controller implements Kubernetes controllers for KubeTask resources
package controller

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubetaskv1alpha1 "github.com/xuezhaojun/kubetask/api/v1alpha1"
)

const (
	// DefaultAgentImage is the default agent container image
	DefaultAgentImage = "quay.io/zhaoxue/kubetask-default-agent:latest"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubetask.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubetask.io,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubetask.io,resources=tasks/finalizers,verbs=update
// +kubebuilder:rbac:groups=kubetask.io,resources=workspaceconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get Task CR
	task := &kubetaskv1alpha1.Task{}
	if err := r.Get(ctx, req.NamespacedName, task); err != nil {
		if errors.IsNotFound(err) {
			// Task deleted, nothing to do
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Task")
		return ctrl.Result{}, err
	}

	// If new, initialize status and create Job
	if task.Status.Phase == "" {
		return r.initializeTask(ctx, task)
	}

	// If completed/failed, skip
	if task.Status.Phase == kubetaskv1alpha1.TaskPhaseSucceeded ||
		task.Status.Phase == kubetaskv1alpha1.TaskPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Update task status from Job status
	if err := r.updateTaskStatusFromJob(ctx, task); err != nil {
		log.Error(err, "unable to update task status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// initializeTask initializes a new Task and creates its Job
func (r *TaskReconciler) initializeTask(ctx context.Context, task *kubetaskv1alpha1.Task) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get agent image from WorkspaceConfig or use default
	agentImage := r.getAgentImage(ctx, task)

	// Generate Job name
	jobName := fmt.Sprintf("%s-job", task.Name)

	// Check if Job already exists
	existingJob := &batchv1.Job{}
	jobKey := types.NamespacedName{Name: jobName, Namespace: task.Namespace}
	if err := r.Get(ctx, jobKey, existingJob); err == nil {
		// Job already exists, update status
		task.Status.JobName = jobName
		task.Status.Phase = kubetaskv1alpha1.TaskPhaseRunning
		now := metav1.Now()
		task.Status.StartTime = &now
		return ctrl.Result{}, r.Status().Update(ctx, task)
	}

	// Create Job with agent image
	job := r.buildJob(task, jobName, agentImage)

	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job", "job", jobName)
		return ctrl.Result{}, err
	}

	// Update status
	task.Status.JobName = jobName
	task.Status.Phase = kubetaskv1alpha1.TaskPhaseRunning
	now := metav1.Now()
	task.Status.StartTime = &now

	if err := r.Status().Update(ctx, task); err != nil {
		log.Error(err, "unable to update Task status")
		return ctrl.Result{}, err
	}

	log.Info("initialized Task", "job", jobName, "image", agentImage)
	return ctrl.Result{}, nil
}

// updateTaskStatusFromJob syncs task status from Job status
func (r *TaskReconciler) updateTaskStatusFromJob(ctx context.Context, task *kubetaskv1alpha1.Task) error {
	log := log.FromContext(ctx)

	if task.Status.JobName == "" {
		return nil
	}

	// Get Job status
	job := &batchv1.Job{}
	jobKey := types.NamespacedName{Name: task.Status.JobName, Namespace: task.Namespace}
	if err := r.Get(ctx, jobKey, job); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Job not found", "job", task.Status.JobName)
			return nil
		}
		return err
	}

	// Check Job completion
	if job.Status.Succeeded > 0 {
		task.Status.Phase = kubetaskv1alpha1.TaskPhaseSucceeded
		now := metav1.Now()
		task.Status.CompletionTime = &now
		log.Info("task succeeded", "job", task.Status.JobName)
		return r.Status().Update(ctx, task)
	} else if job.Status.Failed > 0 {
		task.Status.Phase = kubetaskv1alpha1.TaskPhaseFailed
		now := metav1.Now()
		task.Status.CompletionTime = &now
		log.Info("task failed", "job", task.Status.JobName)
		return r.Status().Update(ctx, task)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubetaskv1alpha1.Task{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// getAgentImage retrieves the agent image from WorkspaceConfig or returns default
func (r *TaskReconciler) getAgentImage(ctx context.Context, task *kubetaskv1alpha1.Task) string {
	log := log.FromContext(ctx)

	// Determine which WorkspaceConfig to use
	configName := "default"
	if task.Spec.WorkspaceConfigRef != "" {
		configName = task.Spec.WorkspaceConfigRef
	}

	// Try to get WorkspaceConfig
	config := &kubetaskv1alpha1.WorkspaceConfig{}
	configKey := types.NamespacedName{
		Name:      configName,
		Namespace: task.Namespace,
	}

	if err := r.Get(ctx, configKey, config); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get WorkspaceConfig, using default image", "workspaceConfig", configName)
		}
		return DefaultAgentImage
	}

	// Use configured image or default
	if config.Spec.AgentImage != "" {
		return config.Spec.AgentImage
	}

	return DefaultAgentImage
}

// buildJob creates a Job object for the task
func (r *TaskReconciler) buildJob(task *kubetaskv1alpha1.Task, jobName, agentImage string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: task.Namespace,
			Labels: map[string]string{
				"app":              "kubetask",
				"kubetask.io/task": task.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: task.APIVersion,
					Kind:       task.Kind,
					Name:       task.Name,
					UID:        task.UID,
					Controller: boolPtr(true),
				},
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "kubetask-agent",
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: agentImage,
							Env: []corev1.EnvVar{
								{
									Name:  "TASK_NAME",
									Value: task.Name,
								},
								{
									Name:  "TASK_NAMESPACE",
									Value: task.Namespace,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

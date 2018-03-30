/*
Copyright The CBI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package buildkit

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
)

type BuildKit struct {
	BuildctlImage string
	BuildkitdAddr string
}

var _ backend.Backend = &BuildKit{}

func (b *BuildKit) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		SupportedLanguageKind: []string{
			crd.LanguageKindDockerfile,
		},
		SupportedContextKind: []string{
			crd.ContextKindGit,
		},
	}
	return res, nil
}

func (b *BuildKit) CreateJobManifest(ctx context.Context, jobName string, buildJob crd.BuildJob) (*batchv1.Job, error) {
	if buildJob.Spec.Push {
		return nil, fmt.Errorf("unsupported Spec.Push: %v", buildJob.Spec.Push)
	}
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if buildJob.Spec.Context.Kind != crd.ContextKindGit {
		return nil, fmt.Errorf("unsupported Spec.Context: %v", buildJob.Spec.Context)
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:  "buildctl-job",
				Image: b.BuildctlImage,
				Command: []string{"buildctl", "--addr", b.BuildkitdAddr,
					"build",
					"--frontend=dockerfile.v0",
					"--frontend-opt", "context=" + buildJob.Spec.Context.GitRef.URL,
					"--no-progress", "--trace", "/dev/stdout",
				},
			},
		},
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: buildJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&buildJob, schema.GroupVersionKind{
					Group:   crd.SchemeGroupVersion.Group,
					Version: crd.SchemeGroupVersion.Version,
					Kind:    "BuildJob",
				}),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}, nil
}
/*
   Copyright 2020 Docker Compose CLI authors

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

package compose

import (
	"context"
	"strings"
	"testing"

	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"

	compose "github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/mocks"
)

func TestDown(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(false)).Return(
		[]moby.Container{
			testContainer("service1", "123", false),
			testContainer("service2", "456", false),
			testContainer("service2", "789", false),
			testContainer("service_orphan", "321", true),
		}, nil)
	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{}, nil)

	// network names are not guaranteed to be unique, ensure Compose handles
	// cleanup properly if duplicates are inadvertently created
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return([]moby.NetworkResource{
			{ID: "abc123", Name: "myProject_default"},
			{ID: "def456", Name: "myProject_default"},
		}, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerStop(gomock.Any(), "456", nil).Return(nil)
	api.EXPECT().ContainerStop(gomock.Any(), "789", nil).Return(nil)

	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true}).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "456", moby.ContainerRemoveOptions{Force: true}).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "789", moby.ContainerRemoveOptions{Force: true}).Return(nil)

	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", "myProject_default")),
	}).Return([]moby.NetworkResource{
		{ID: "abc123", Name: "myProject_default"},
		{ID: "def456", Name: "myProject_default"},
	}, nil)
	api.EXPECT().NetworkRemove(gomock.Any(), "abc123").Return(nil)
	api.EXPECT().NetworkRemove(gomock.Any(), "def456").Return(nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{})
	assert.NilError(t, err)
}

func TestDownRemoveOrphans(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(true)).Return(
		[]moby.Container{
			testContainer("service1", "123", false),
			testContainer("service2", "789", false),
			testContainer("service_orphan", "321", true),
		}, nil)
	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{}, nil)
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return([]moby.NetworkResource{{Name: "myProject_default"}}, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerStop(gomock.Any(), "789", nil).Return(nil)
	api.EXPECT().ContainerStop(gomock.Any(), "321", nil).Return(nil)

	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true}).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "789", moby.ContainerRemoveOptions{Force: true}).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "321", moby.ContainerRemoveOptions{Force: true}).Return(nil)

	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", "myProject_default")),
	}).Return([]moby.NetworkResource{{ID: "abc123", Name: "myProject_default"}}, nil)
	api.EXPECT().NetworkRemove(gomock.Any(), "abc123").Return(nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{RemoveOrphans: true})
	assert.NilError(t, err)
}

func TestDownRemoveVolumes(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(false)).Return(
		[]moby.Container{testContainer("service1", "123", false)}, nil)
	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{
			Volumes: []*moby.Volume{{Name: "myProject_volume"}},
		}, nil)
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return(nil, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	api.EXPECT().VolumeRemove(gomock.Any(), "myProject_volume", true).Return(nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{Volumes: true})
	assert.NilError(t, err)
}

func TestDownRemoveImageLocal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	container := testContainer("service1", "123", false)
	container.Labels[compose.ImageNameLabel] = ""

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(false)).Return(
		[]moby.Container{container}, nil)

	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{
			Volumes: []*moby.Volume{{Name: "myProject_volume"}},
		}, nil)
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return(nil, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true}).Return(nil)

	api.EXPECT().ImageRemove(gomock.Any(), "testproject-service1", moby.ImageRemoveOptions{}).Return(nil, nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{Images: "local"})
	assert.NilError(t, err)
}

func TestDownRemoveImageLocalNoLabel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	container := testContainer("service1", "123", false)

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(false)).Return(
		[]moby.Container{container}, nil)

	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{
			Volumes: []*moby.Volume{{Name: "myProject_volume"}},
		}, nil)
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return(nil, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true}).Return(nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{Images: "local"})
	assert.NilError(t, err)
}

func TestDownRemoveImageAll(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	api := mocks.NewMockAPIClient(mockCtrl)
	cli := mocks.NewMockCli(mockCtrl)
	tested := composeService{
		dockerCli: cli,
	}
	cli.EXPECT().Client().Return(api).AnyTimes()

	api.EXPECT().ContainerList(gomock.Any(), projectFilterListOpt(false)).Return(
		[]moby.Container{testContainer("service1", "123", false)}, nil)

	api.EXPECT().VolumeList(gomock.Any(), filters.NewArgs(projectFilter(strings.ToLower(testProject)))).
		Return(volume.VolumeListOKBody{
			Volumes: []*moby.Volume{{Name: "myProject_volume"}},
		}, nil)
	api.EXPECT().NetworkList(gomock.Any(), moby.NetworkListOptions{Filters: filters.NewArgs(projectFilter(strings.ToLower(testProject)))}).
		Return(nil, nil)

	api.EXPECT().ContainerStop(gomock.Any(), "123", nil).Return(nil)
	api.EXPECT().ContainerRemove(gomock.Any(), "123", moby.ContainerRemoveOptions{Force: true}).Return(nil)

	api.EXPECT().ImageRemove(gomock.Any(), "service1-img", moby.ImageRemoveOptions{}).Return(nil, nil)

	err := tested.Down(context.Background(), strings.ToLower(testProject), compose.DownOptions{Images: "all"})
	assert.NilError(t, err)
}

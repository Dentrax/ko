// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package publish

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/ko/pkg/build"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type LayoutPublisher struct {
	p    layout.Path
	tags []string
}

// NewLayout returns a new publish.Interface that saves images to an OCI Image Layout.
func NewLayout(path string, tags []string) (Interface, error) {
	p, err := layout.FromPath(path)
	if err != nil {
		p, err = layout.Write(path, empty.Index)
		if err != nil {
			return nil, err
		}
	}
	if len(tags) == 0 {
		tags = []string{"latest"}
	}
	return &LayoutPublisher{p, tags}, nil
}

func (l *LayoutPublisher) writeResult(br build.Result) error {
	mt, err := br.MediaType()
	if err != nil {
		return err
	}

	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		idx, ok := br.(v1.ImageIndex)
		if !ok {
			return fmt.Errorf("failed to interpret result as index: %v", br)
		}
		for _, t := range l.tags {
			if err := l.p.AppendIndex(idx,
				layout.WithAnnotations(map[string]string{
					specsv1.AnnotationRefName: t,
				}),
			); err != nil {
				return err
			}
		}
		return nil
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		img, ok := br.(v1.Image)
		if !ok {
			return fmt.Errorf("failed to interpret result as image: %v", br)
		}
		for _, t := range l.tags {
			if err := l.p.AppendImage(img,
				layout.WithAnnotations(map[string]string{
					specsv1.AnnotationRefName: t,
				}),
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("result image media type: %s", mt)
	}
}

// Publish implements publish.Interface.
func (l *LayoutPublisher) Publish(_ context.Context, br build.Result, s string) (name.Reference, error) {
	log.Printf("Saving %v", s)
	if err := l.writeResult(br); err != nil {
		return nil, err
	}
	log.Printf("Saved %v", s)

	h, err := br.Digest()
	if err != nil {
		return nil, err
	}

	dig, err := name.NewDigest(fmt.Sprintf("%s@%s", l.p, h))
	if err != nil {
		return nil, err
	}

	return dig, nil
}

func (l *LayoutPublisher) Close() error {
	return nil
}

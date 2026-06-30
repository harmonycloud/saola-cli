/*
Copyright 2025 The OpenSaola Authors.

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

package images

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportWithSkopeo_BuildsCopyArgs(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}
	targets := []ImportTarget{{
		Name:        "milvus:v2.5.8",
		OCITag:      "milvus-v2.5.8",
		Destination: "10.10.101.172:443/middleware/milvus:v2.5.8",
	}}

	err := importWithSkopeo(context.Background(), runner, "/tmp/oci/images", targets, ImportOptions{
		Platform: "linux/amd64",
		Insecure: true,
		Creds:    "user:password",
	}, io.Discard)
	if err != nil {
		t.Fatalf("importWithSkopeo: %v", err)
	}
	if len(runner.runs) != 1 {
		t.Fatalf("expected one skopeo copy run, got %#v", runner.runs)
	}
	if runner.runs[0].name != toolSkopeo {
		t.Fatalf("expected skopeo command, got %q", runner.runs[0].name)
	}
	got := strings.Join(runner.runs[0].args, " ")
	for _, want := range []string{
		"copy",
		"--override-os linux",
		"--override-arch amd64",
		"--dest-tls-verify=false",
		"--dest-creds user:password",
		"oci:/tmp/oci/images:milvus-v2.5.8",
		"docker://10.10.101.172:443/middleware/milvus:v2.5.8",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected args to contain %q, got %q", want, got)
		}
	}
}

func TestImportWithSkopeo_MultiArchKeepsAll(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}

	err := importWithSkopeo(context.Background(), runner, "/tmp/oci/images", []ImportTarget{{
		Name:        "etcd:3.5.18-r1",
		OCITag:      "etcd-3.5.18-r1",
		Destination: "dst/middleware/etcd:3.5.18-r1",
	}}, ImportOptions{Platform: "all", MultiArch: true}, io.Discard)
	if err != nil {
		t.Fatalf("importWithSkopeo: %v", err)
	}
	got := strings.Join(runner.runs[0].args, " ")
	if !strings.Contains(got, "--all") {
		t.Errorf("expected --all for multi-arch import, got %q", got)
	}
	if strings.Contains(got, "--override-os") {
		t.Errorf("did not expect platform override when platform=all, got %q", got)
	}
}

func TestImportWithSkopeo_OmitsCredsAndTLSWhenUnset(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}

	err := importWithSkopeo(context.Background(), runner, "/tmp/oci/images", []ImportTarget{{
		Name:        "attu:v2.5.3",
		OCITag:      "attu-v2.5.3",
		Destination: "dst/middleware/attu:v2.5.3",
	}}, ImportOptions{Platform: "all"}, io.Discard)
	if err != nil {
		t.Fatalf("importWithSkopeo: %v", err)
	}
	got := strings.Join(runner.runs[0].args, " ")
	if strings.Contains(got, "--dest-creds") {
		t.Errorf("did not expect --dest-creds without credentials, got %q", got)
	}
	if strings.Contains(got, "--dest-tls-verify=false") {
		t.Errorf("did not expect --dest-tls-verify=false without --insecure, got %q", got)
	}
}

func TestImportImageNames_PrefersLockFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder; lock takes precedence")
	writeFile(t, archive+".lock.json", `{"images":[{"name":"milvus:v2.5.8"},{"name":"etcd:3.5.18-r1"},{"name":"milvus:v2.5.8"}]}`)

	names, err := importImageNames(ImportOptions{Archive: archive})
	if err != nil {
		t.Fatalf("importImageNames: %v", err)
	}
	if strings.Join(names, ",") != "milvus:v2.5.8,etcd:3.5.18-r1" {
		t.Fatalf("expected deduped lock names, got %v", names)
	}
}

func TestImportImageNames_ExplicitMissingLockIsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder")

	_, err := importImageNames(ImportOptions{Archive: archive, LockFile: filepath.Join(dir, "nope.lock.json")})
	if err == nil || !strings.Contains(err.Error(), "lock file") {
		t.Fatalf("expected explicit lock read error, got %v", err)
	}
}

func TestImportImageNames_ImplicitMissingLockIsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder")

	_, err := importImageNames(ImportOptions{Archive: archive})
	if err == nil || !strings.Contains(err.Error(), "lock file") {
		t.Fatalf("expected implicit lock read error, got %v", err)
	}
}

func TestImportPackage_DryRunListsTargets(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder")
	writeFile(t, archive+".lock.json", `{"images":[{"name":"milvus:v2.5.8"},{"name":"attu:v2.5.3"}]}`)

	result, err := ImportPackage(context.Background(), ImportOptions{
		Archive:    archive,
		Repository: "10.10.101.172:443/middleware/", // trailing slash should be trimmed
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("ImportPackage dry-run: %v", err)
	}
	if !result.DryRun {
		t.Fatal("expected DryRun result")
	}
	if len(result.Targets) != 2 {
		t.Fatalf("expected two targets, got %#v", result.Targets)
	}
	if result.Targets[0].Destination != "10.10.101.172:443/middleware/milvus:v2.5.8" {
		t.Errorf("unexpected destination: %q", result.Targets[0].Destination)
	}
	if result.Targets[0].OCITag != "milvus-v2.5.8" {
		t.Errorf("unexpected oci tag: %q", result.Targets[0].OCITag)
	}
}

func TestImportPackage_RequiresRepository(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder")

	_, err := ImportPackage(context.Background(), ImportOptions{Archive: archive})
	if err == nil || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestImportPackage_MissingArchiveIsError(t *testing.T) {
	t.Parallel()
	_, err := ImportPackage(context.Background(), ImportOptions{
		Archive:    filepath.Join(t.TempDir(), "does-not-exist.tar"),
		Repository: "dst/middleware",
	})
	if err == nil || !strings.Contains(err.Error(), "archive") {
		t.Fatalf("expected archive error, got %v", err)
	}
}

func TestImportPackage_MissingSkopeoFailsClearly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archive := filepath.Join(dir, "images.tar")
	writeFile(t, archive, "placeholder")
	writeFile(t, archive+".lock.json", `{"images":[{"name":"milvus:v2.5.8"}]}`)

	_, err := ImportPackage(context.Background(), ImportOptions{
		Archive:    archive,
		Repository: "dst/middleware",
		Runner:     &fakeRunner{tools: map[string]bool{}}, // skopeo absent
	})
	if err == nil || !strings.Contains(err.Error(), "skopeo") {
		t.Fatalf("expected skopeo-required error, got %v", err)
	}
}

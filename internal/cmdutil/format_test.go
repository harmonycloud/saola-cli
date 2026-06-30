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

package cmdutil

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFormatAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"Days", 72 * time.Hour, "3d"},
		{"Hours", 5 * time.Hour, "5h"},
		{"Minutes", 45 * time.Minute, "45m"},
		{"Seconds", 10 * time.Second, "10s"},
		{"Negative", -1 * time.Second, "<unknown>"},
		{"Zero", 0, "0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatAge(tt.d)
			if got != tt.want {
				t.Errorf("FormatAge(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"Short", "hi", 10, "hi"},
		{"Exact", "hello", 5, "hello"},
		{"Long", "hello world", 8, "hello..."},
		{"Max3", "hello", 3, "hel"},
		{"UTF8", "字段路径错误：spec.necessary.resource.etcd.volume", 8, "字段路径错..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Truncate(%q, %d) returned invalid UTF-8: %q", tt.s, tt.max, got)
			}
		})
	}
}

func TestFormatLabelsShort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		labels map[string]string
		keys   []string
		want   string
	}{
		{
			"Normal",
			map[string]string{"app": "redis", "version": "7.0"},
			[]string{"app", "version"},
			"redis/7.0",
		},
		{
			"MissingKey",
			map[string]string{"app": "redis"},
			[]string{"app", "version"},
			"redis",
		},
		{
			"Empty",
			map[string]string{},
			[]string{"app"},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatLabelsShort(tt.labels, tt.keys)
			if got != tt.want {
				t.Errorf("FormatLabelsShort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintDiagnostics_PrintsFullConditionMessage(t *testing.T) {
	t.Parallel()

	fullMessage := "phase=workload-readiness; failedObject=v1/Pod middleware-operator/opensaola-install-crds; causeCategory=RegistryTLS; cause=ErrImagePull: failed to pull image \"10.10.101.172:443/middleware/kubectl:v1.30.14\": x509: certificate signed by unknown authority; next=kubectl describe pod opensaola-install-crds -n middleware-operator"
	var out bytes.Buffer
	PrintDiagnostics(&out, DiagnosticsOptions{
		StatusReason: "operator unavailable",
		Runtime:      fullMessage,
		Conditions: []metav1.Condition{{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ImagePullBackOff",
			Message: fullMessage,
		}},
	})

	got := out.String()
	for _, want := range []string{
		"Diagnostics:",
		"Runtime:",
		"Status Reason:",
		"Conditions:",
		fullMessage,
		"x509: certificate signed by unknown authority",
		"10.10.101.172:443/middleware/kubectl:v1.30.14",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diagnostics output to contain %q, got %q", want, got)
		}
	}
}

func TestCollectObjectEvents_FiltersByUIDAndPrintsFullRecentEvents(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	ts := metav1.NewTime(time.Date(2026, 6, 29, 10, 13, 1, 0, time.UTC))
	fullMessage := "failed to pull image \"10.10.101.172:443/middleware/kubectl:v1.30.14\": x509: certificate signed by unknown authority"
	matching := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "pull-failed", Namespace: "middleware-operator"},
		Type:       corev1.EventTypeWarning,
		Reason:     "FailedPull",
		Message:    fullMessage,
		Count:      3,
		InvolvedObject: corev1.ObjectReference{
			Kind:      "MiddlewareOperator",
			Namespace: "middleware-operator",
			Name:      "opensaola",
			UID:       types.UID("mo-uid"),
		},
		LastTimestamp: ts,
	}
	unrelated := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "middleware-operator"},
		Type:       corev1.EventTypeWarning,
		Reason:     "FailedPull",
		Message:    "unrelated",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "MiddlewareOperator",
			Namespace: "middleware-operator",
			Name:      "opensaola",
			UID:       types.UID("other-uid"),
		},
		LastTimestamp: ts,
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(matching, unrelated).Build()

	events, err := CollectObjectEvents(context.Background(), cli, EventObjectRef{
		Kind:      "MiddlewareOperator",
		Namespace: "middleware-operator",
		Name:      "opensaola",
		UID:       types.UID("mo-uid"),
	}, 10)
	if err != nil {
		t.Fatalf("CollectObjectEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 matching event, got %d", len(events))
	}

	var out bytes.Buffer
	PrintDiagnostics(&out, DiagnosticsOptions{RecentEvents: events})
	got := out.String()
	for _, want := range []string{
		"Recent Events:",
		"2026-06-29T10:13:01Z Warning FailedPull count=3 involvedObject=MiddlewareOperator/middleware-operator/opensaola",
		fullMessage,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diagnostics output to contain %q, got %q", want, got)
		}
	}
	if strings.Contains(got, "unrelated") {
		t.Fatalf("expected unrelated event to be filtered out, got %q", got)
	}
}

func TestCollectObjectEventsForRefs_IncludesFailedObjectFromDiagnostics(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	ts := metav1.NewTime(time.Date(2026, 6, 29, 10, 13, 1, 0, time.UTC))
	eventMessage := "failed to pull image: x509: certificate signed by unknown authority"
	podEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-pull-failed", Namespace: "middleware-operator"},
		Type:       corev1.EventTypeWarning,
		Reason:     "FailedPull",
		Message:    eventMessage,
		Count:      4,
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Namespace: "middleware-operator",
			Name:      "opensaola-install-crds",
		},
		LastTimestamp: ts,
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(podEvent).Build()

	diagnostic := "phase=workload-readiness; failedObject=v1/Pod middleware-operator/opensaola-install-crds; causeCategory=RegistryTLS"
	refs := []EventObjectRef{{
		Kind:      "MiddlewareOperator",
		Namespace: "middleware-operator",
		Name:      "opensaola",
		UID:       types.UID("mo-uid"),
	}}
	refs = append(refs, DiagnosticObjectEventRefs("middleware-operator", diagnostic)...)
	events, err := CollectObjectEventsForRefs(context.Background(), cli, refs, 10)
	if err != nil {
		t.Fatalf("CollectObjectEventsForRefs returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 diagnostic failedObject event, got %d", len(events))
	}

	var out bytes.Buffer
	PrintDiagnostics(&out, DiagnosticsOptions{RecentEvents: events})
	got := out.String()
	for _, want := range []string{
		"Recent Events:",
		"involvedObject=Pod/middleware-operator/opensaola-install-crds",
		eventMessage,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diagnostics output to contain %q, got %q", want, got)
		}
	}
}

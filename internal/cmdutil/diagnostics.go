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
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DiagnosticsOptions struct {
	StatusReason         string
	Runtime              string
	CustomResourceReason string
	Conditions           []metav1.Condition
	RecentEvents         []corev1.Event
	EventsError          error
}

type EventObjectRef struct {
	Kind      string
	Namespace string
	Name      string
	UID       types.UID
}

func CollectObjectEvents(ctx context.Context, cli client.Client, ref EventObjectRef, limit int) ([]corev1.Event, error) {
	return CollectObjectEventsForRefs(ctx, cli, []EventObjectRef{ref}, limit)
}

func CollectObjectEventsForRefs(ctx context.Context, cli client.Client, refs []EventObjectRef, limit int) ([]corev1.Event, error) {
	refs = normalizeEventObjectRefs(refs)
	if cli == nil || len(refs) == 0 {
		return nil, nil
	}

	refsByNamespace := make(map[string][]EventObjectRef)
	for _, ref := range refs {
		refsByNamespace[ref.Namespace] = append(refsByNamespace[ref.Namespace], ref)
	}

	events := make([]corev1.Event, 0)
	seen := make(map[string]struct{})
	for namespace, namespaceRefs := range refsByNamespace {
		var eventList corev1.EventList
		if err := cli.List(ctx, &eventList, client.InNamespace(namespace)); err != nil {
			return nil, fmt.Errorf("list events in namespace %s: %w", namespace, err)
		}
		for _, event := range eventList.Items {
			if !eventMatchesAnyObject(event, namespaceRefs) || !isDiagnosticEvent(event) {
				continue
			}
			key := event.Namespace + "/" + event.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			events = append(events, event)
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return eventTimestamp(events[i]).After(eventTimestamp(events[j]))
	})
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func DiagnosticObjectEventRefs(defaultNamespace string, diagnostics ...string) []EventObjectRef {
	refs := make([]EventObjectRef, 0)
	for _, diagnostic := range diagnostics {
		refs = append(refs, diagnosticObjectEventRefs(defaultNamespace, diagnostic)...)
	}
	return normalizeEventObjectRefs(refs)
}

func DiagnosticObjectEventRefsFromConditions(defaultNamespace string, conditions []metav1.Condition) []EventObjectRef {
	refs := make([]EventObjectRef, 0, len(conditions))
	for _, condition := range conditions {
		refs = append(refs, DiagnosticObjectEventRefs(defaultNamespace, condition.Message)...)
	}
	return normalizeEventObjectRefs(refs)
}

func PrintDiagnostics(w io.Writer, opts DiagnosticsOptions) {
	conditionDiagnostics := diagnosticConditions(opts.Conditions)
	if opts.StatusReason == "" && opts.Runtime == "" && opts.CustomResourceReason == "" && len(conditionDiagnostics) == 0 && len(opts.RecentEvents) == 0 && opts.EventsError == nil {
		return
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Diagnostics:")
	if opts.Runtime != "" {
		fmt.Fprintln(w, "  Runtime:")
		printIndentedBlock(w, opts.Runtime, "    ")
	}
	if opts.StatusReason != "" {
		fmt.Fprintln(w, "  Status Reason:")
		printIndentedBlock(w, opts.StatusReason, "    ")
	}
	if opts.CustomResourceReason != "" && opts.CustomResourceReason != opts.StatusReason {
		fmt.Fprintln(w, "  CustomResources Reason:")
		printIndentedBlock(w, opts.CustomResourceReason, "    ")
	}
	if len(conditionDiagnostics) > 0 {
		fmt.Fprintln(w, "  Conditions:")
		for _, condition := range conditionDiagnostics {
			fmt.Fprintf(w, "    - Type: %s\n", condition.Type)
			fmt.Fprintf(w, "      Status: %s\n", condition.Status)
			if condition.Reason != "" {
				fmt.Fprintf(w, "      Reason: %s\n", condition.Reason)
			}
			if !condition.LastTransitionTime.IsZero() {
				fmt.Fprintf(w, "      LastTransitionTime: %s\n", condition.LastTransitionTime.Format("2006-01-02T15:04:05Z"))
			}
			if condition.Message != "" {
				fmt.Fprintln(w, "      Message:")
				printIndentedBlock(w, condition.Message, "        ")
			}
		}
	}
	if opts.EventsError != nil {
		fmt.Fprintln(w, "  Recent Events:")
		fmt.Fprintf(w, "    unavailable: %v\n", opts.EventsError)
	} else if len(opts.RecentEvents) > 0 {
		fmt.Fprintln(w, "  Recent Events:")
		for _, event := range opts.RecentEvents {
			fmt.Fprintf(w, "    - %s %s %s count=%d involvedObject=%s/%s/%s\n",
				eventTimestamp(event).UTC().Format("2006-01-02T15:04:05Z"),
				event.Type,
				event.Reason,
				event.Count,
				event.InvolvedObject.Kind,
				eventInvolvedNamespace(event),
				event.InvolvedObject.Name,
			)
			if event.Message != "" {
				fmt.Fprintln(w, "      Message:")
				printIndentedBlock(w, event.Message, "        ")
			}
		}
	}
}

func diagnosticConditions(conditions []metav1.Condition) []metav1.Condition {
	result := make([]metav1.Condition, 0, len(conditions))
	for _, condition := range conditions {
		if condition.Message == "" {
			continue
		}
		if condition.Status != metav1.ConditionTrue || strings.Contains(condition.Message, "phase=") || len([]rune(condition.Message)) > 60 {
			result = append(result, condition)
		}
	}
	return result
}

func diagnosticObjectEventRefs(defaultNamespace, diagnostic string) []EventObjectRef {
	refs := make([]EventObjectRef, 0)
	for {
		idx := strings.Index(diagnostic, "failedObject=")
		if idx < 0 {
			return refs
		}
		diagnostic = diagnostic[idx+len("failedObject="):]
		value := diagnostic
		if end := strings.Index(value, ";"); end >= 0 {
			value = value[:end]
			diagnostic = diagnostic[end:]
		} else {
			diagnostic = ""
		}
		if ref, ok := parseDiagnosticObjectRef(defaultNamespace, strings.TrimSpace(value)); ok {
			refs = append(refs, ref)
		}
	}
}

func parseDiagnosticObjectRef(defaultNamespace, value string) (EventObjectRef, bool) {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return EventObjectRef{}, false
	}
	kind := fields[0]
	if idx := strings.LastIndex(kind, "/"); idx >= 0 {
		kind = kind[idx+1:]
	}
	if kind == "" || strings.HasPrefix(kind, "<") {
		return EventObjectRef{}, false
	}

	namespace := defaultNamespace
	name := fields[1]
	if idx := strings.Index(name, "/"); idx >= 0 {
		namespace = name[:idx]
		name = name[idx+1:]
	}
	if namespace == "" || name == "" {
		return EventObjectRef{}, false
	}
	return EventObjectRef{Kind: kind, Namespace: namespace, Name: name}, true
}

func normalizeEventObjectRefs(refs []EventObjectRef) []EventObjectRef {
	normalized := make([]EventObjectRef, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.Namespace == "" || ref.Name == "" || ref.Kind == "" {
			continue
		}
		key := ref.Kind + "/" + ref.Namespace + "/" + ref.Name + "/" + string(ref.UID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, ref)
	}
	return normalized
}

func printIndentedBlock(w io.Writer, text, prefix string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "%s%s\n", prefix, line)
	}
}

func eventMatchesObject(event corev1.Event, ref EventObjectRef) bool {
	if event.Namespace != ref.Namespace {
		return false
	}
	if ref.UID != "" && event.InvolvedObject.UID != "" {
		return event.InvolvedObject.UID == ref.UID
	}
	return event.InvolvedObject.Kind == ref.Kind &&
		event.InvolvedObject.Name == ref.Name &&
		(event.InvolvedObject.Namespace == "" || event.InvolvedObject.Namespace == ref.Namespace)
}

func eventMatchesAnyObject(event corev1.Event, refs []EventObjectRef) bool {
	for _, ref := range refs {
		if eventMatchesObject(event, ref) {
			return true
		}
	}
	return false
}

func eventInvolvedNamespace(event corev1.Event) string {
	if event.InvolvedObject.Namespace != "" {
		return event.InvolvedObject.Namespace
	}
	return event.Namespace
}

func isDiagnosticEvent(event corev1.Event) bool {
	if event.Type == corev1.EventTypeWarning {
		return true
	}
	switch event.Reason {
	case "BackOff", "ErrImagePull", "Failed", "FailedAttachVolume", "FailedBinding", "FailedCreate", "FailedMount", "FailedPull", "FailedScheduling", "ImagePullBackOff", "Unhealthy":
		return true
	default:
		return false
	}
}

func eventTimestamp(event corev1.Event) time.Time {
	switch {
	case !event.EventTime.IsZero():
		return event.EventTime.Time
	case !event.LastTimestamp.IsZero():
		return event.LastTimestamp.Time
	case !event.FirstTimestamp.IsZero():
		return event.FirstTimestamp.Time
	default:
		return event.CreationTimestamp.Time
	}
}

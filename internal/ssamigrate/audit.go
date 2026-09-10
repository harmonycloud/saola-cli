// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kjson "sigs.k8s.io/json"
)

type AuditEvent struct {
	AuditID    string `json:"auditID"`
	Stage      string `json:"stage"`
	Verb       string `json:"verb"`
	RequestURI string `json:"requestURI"`
	UserAgent  string `json:"userAgent"`
	User       struct {
		Username string `json:"username"`
	} `json:"user"`
	ImpersonatedUser json.RawMessage `json:"impersonatedUser"`
	ResponseStatus   struct {
		Code int `json:"code"`
	} `json:"responseStatus"`
	ObjectRef struct {
		UID         string `json:"uid"`
		Subresource string `json:"subresource"`
	} `json:"objectRef"`
	RequestObject  *unstructured.Unstructured `json:"requestObject"`
	ResponseObject *unstructured.Unstructured `json:"responseObject"`
}

func (e AuditEvent) manager() string {
	u, err := url.Parse(e.RequestURI)
	if err == nil {
		if v := u.Query().Get("fieldManager"); v != "" {
			return v
		}
	}
	return strings.Split(e.UserAgent, "/")[0]
}
func ReadAudit(r io.Reader) ([]AuditEvent, error) {
	events := []AuditEvent{}
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 64*1024), 32*1024*1024)
	line := 0
	for scan.Scan() {
		line++
		if len(strings.TrimSpace(scan.Text())) == 0 {
			continue
		}
		var e AuditEvent
		if err := json.Unmarshal(scan.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("audit line %d: %w", line, err)
		}
		events = append(events, e)
	}
	return events, scan.Err()
}
func creation(events []AuditEvent, live *unstructured.Unstructured, writer string) (*AuditEvent, error) {
	if writer == "" {
		return nil, fmt.Errorf("trusted writer username required")
	}
	var found *AuditEvent
	for i := range events {
		e := &events[i]
		if e.Stage != "ResponseComplete" || e.Verb != "create" || e.ResponseStatus.Code != 201 || e.ResponseObject == nil || e.ResponseObject.GetUID() != live.GetUID() {
			continue
		}
		if e.User.Username != writer || len(e.ImpersonatedUser) > 0 && string(e.ImpersonatedUser) != "null" {
			return nil, fmt.Errorf("create identity does not match trusted writer")
		}
		if found != nil {
			return nil, fmt.Errorf("duplicate create audit events")
		}
		found = e
	}
	if found == nil || found.RequestObject == nil || found.AuditID == "" || found.manager() == "" {
		return nil, fmt.Errorf("complete successful RequestResponse create audit required")
	}
	for _, o := range []*unstructured.Unstructured{found.RequestObject, found.ResponseObject} {
		if o.GetAPIVersion() != live.GetAPIVersion() || o.GetKind() != live.GetKind() || o.GetName() != live.GetName() || o.GetNamespace() != live.GetNamespace() {
			return nil, fmt.Errorf("audit object identity/API version mismatch")
		}
	}
	for _, e := range events {
		if e.Stage != "ResponseComplete" || e.ResponseStatus.Code < 200 || e.ResponseStatus.Code >= 300 || e.ObjectRef.Subresource != "" || e.Verb == "get" || e.Verb == "list" || e.Verb == "watch" || e.Verb == "create" {
			continue
		}
		uid := e.ObjectRef.UID
		if e.ResponseObject != nil {
			uid = string(e.ResponseObject.GetUID())
		}
		if uid == string(live.GetUID()) && e.manager() == found.manager() && (e.User.Username != writer || len(e.ImpersonatedUser) > 0 && string(e.ImpersonatedUser) != "null") {
			return nil, fmt.Errorf("historical manager reused by another writer")
		}
	}
	return found, nil
}

// UnmarshalJSON tolerates JSON Patch arrays in unrelated audit events.
// UnmarshalJSON 允许其他审计事件包含 JSON Patch 数组请求体。
func (e *AuditEvent) UnmarshalJSON(data []byte) error {
	type plain AuditEvent
	var wire struct {
		*plain
		Request json.RawMessage `json:"requestObject"`
	}
	wire.plain = (*plain)(e)
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if len(wire.Request) > 0 && wire.Request[0] == '{' {
		e.RequestObject = &unstructured.Unstructured{}
		return kjson.UnmarshalCaseSensitivePreserveInts(wire.Request, &e.RequestObject.Object)
	}
	return nil
}

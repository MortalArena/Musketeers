package agent_bridge

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
)

func TestTaskProtocol_CreateTaskRequest(t *testing.T) {
	tp := NewTaskProtocol()

	payload := map[string]interface{}{
		"key": "value",
	}

	req, err := tp.CreateTaskRequest("execute", payload, 1, 30)
	if err != nil {
		t.Fatalf("CreateTaskRequest failed: %v", err)
	}

	if req.TaskID == "" {
		t.Fatal("Expected non-empty TaskID")
	}

	if req.Type != "execute" {
		t.Errorf("Expected Type execute, got %s", req.Type)
	}

	if req.Priority != 1 {
		t.Errorf("Expected Priority 1, got %d", req.Priority)
	}

	if req.TimeoutSec != 30 {
		t.Errorf("Expected TimeoutSec 30, got %d", req.TimeoutSec)
	}
}

func TestTaskProtocol_CreateTaskResponse(t *testing.T) {
	tp := NewTaskProtocol()

	result := map[string]interface{}{
		"key": "value",
	}

	resp := tp.CreateTaskResponse("task-123", true, result, "")

	if resp.TaskID != "task-123" {
		t.Errorf("Expected TaskID task-123, got %s", resp.TaskID)
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}
}

func TestTaskProtocol_EncodeTaskRequest(t *testing.T) {
	tp := NewTaskProtocol()

	payload := map[string]interface{}{
		"key": "value",
	}

	req, err := tp.CreateTaskRequest("execute", payload, 1, 30)
	if err != nil {
		t.Fatalf("CreateTaskRequest failed: %v", err)
	}

	msg, err := tp.EncodeTaskRequest(req)
	if err != nil {
		t.Fatalf("EncodeTaskRequest failed: %v", err)
	}

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	if msg.Type != protocol.MessageTypeTaskRequest {
		t.Errorf("Expected MessageTypeTaskRequest, got %s", msg.Type)
	}
}

func TestTaskProtocol_DecodeTaskRequest(t *testing.T) {
	tp := NewTaskProtocol()

	payload := map[string]interface{}{
		"key": "value",
	}

	req, err := tp.CreateTaskRequest("execute", payload, 1, 30)
	if err != nil {
		t.Fatalf("CreateTaskRequest failed: %v", err)
	}

	msg, err := tp.EncodeTaskRequest(req)
	if err != nil {
		t.Fatalf("EncodeTaskRequest failed: %v", err)
	}

	decoded, err := tp.DecodeTaskRequest(msg)
	if err != nil {
		t.Fatalf("DecodeTaskRequest failed: %v", err)
	}

	if decoded.TaskID != req.TaskID {
		t.Errorf("Expected TaskID %s, got %s", req.TaskID, decoded.TaskID)
	}

	if decoded.Type != req.Type {
		t.Errorf("Expected Type %s, got %s", req.Type, decoded.Type)
	}
}

func TestTaskProtocol_EncodeTaskResponse(t *testing.T) {
	tp := NewTaskProtocol()

	result := map[string]interface{}{
		"key": "value",
	}

	resp := tp.CreateTaskResponse("task-123", true, result, "")

	msg, err := tp.EncodeTaskResponse(resp)
	if err != nil {
		t.Fatalf("EncodeTaskResponse failed: %v", err)
	}

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	if msg.Type != protocol.MessageTypeTaskResponse {
		t.Errorf("Expected MessageTypeTaskResponse, got %s", msg.Type)
	}
}

func TestTaskProtocol_DecodeTaskResponse(t *testing.T) {
	tp := NewTaskProtocol()

	result := map[string]interface{}{
		"key": "value",
	}

	resp := tp.CreateTaskResponse("task-123", true, result, "")

	msg, err := tp.EncodeTaskResponse(resp)
	if err != nil {
		t.Fatalf("EncodeTaskResponse failed: %v", err)
	}

	decoded, err := tp.DecodeTaskResponse(msg)
	if err != nil {
		t.Fatalf("DecodeTaskResponse failed: %v", err)
	}

	if decoded.TaskID != resp.TaskID {
		t.Errorf("Expected TaskID %s, got %s", resp.TaskID, decoded.TaskID)
	}

	if decoded.Success != resp.Success {
		t.Errorf("Expected Success %v, got %v", resp.Success, decoded.Success)
	}
}

func TestTaskProtocol_ValidateTaskRequest(t *testing.T) {
	tp := NewTaskProtocol()

	payload := map[string]interface{}{
		"key": "value",
	}

	req, err := tp.CreateTaskRequest("execute", payload, 1, 30)
	if err != nil {
		t.Fatalf("CreateTaskRequest failed: %v", err)
	}

	err = tp.ValidateTaskRequest(req)
	if err != nil {
		t.Fatalf("ValidateTaskRequest failed: %v", err)
	}
}

func TestTaskProtocol_ValidateTaskRequest_Invalid(t *testing.T) {
	tp := NewTaskProtocol()

	// طلب بدون TaskID
	req := &TaskRequest{
		Type:       "execute",
		Priority:   1,
		TimeoutSec: 30,
	}

	err := tp.ValidateTaskRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid task request")
	}
}

func TestTaskProtocol_ValidateTaskResponse(t *testing.T) {
	tp := NewTaskProtocol()

	result := map[string]interface{}{
		"key": "value",
	}

	resp := tp.CreateTaskResponse("task-123", true, result, "")

	err := tp.ValidateTaskResponse(resp)
	if err != nil {
		t.Fatalf("ValidateTaskResponse failed: %v", err)
	}
}

func TestTaskProtocol_ValidateTaskResponse_Invalid(t *testing.T) {
	tp := NewTaskProtocol()

	// استجابة فاشلة بدون رسالة خطأ
	resp := &TaskResponse{
		TaskID:  "task-123",
		Success: false,
	}

	err := tp.ValidateTaskResponse(resp)
	if err == nil {
		t.Fatal("Expected error for invalid task response")
	}
}

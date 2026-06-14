package agent_bridge

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
	"github.com/sirupsen/logrus"
)

func TestMultiplexedBridge_Send(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	msg := protocol.NewMessage(protocol.MessageTypeTaskRequest, []byte("test-payload"))

	err := mb.Send(LaneWorkflow, msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestMultiplexedBridge_Send_InvalidLane(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	msg := protocol.NewMessage(protocol.MessageTypeTaskRequest, []byte("test-payload"))

	// استخدام LaneType غير صالح
	err := mb.Send(LaneType(99), msg)
	if err == nil {
		t.Fatal("Expected error for invalid lane type")
	}
}

func TestMultiplexedBridge_Receive(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	msg := protocol.NewMessage(protocol.MessageTypeTaskRequest, []byte("test-payload"))

	err := mb.Send(LaneWorkflow, msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	received, err := mb.Receive(LaneWorkflow)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if received == nil {
		t.Fatal("Expected non-nil message")
	}
}

func TestMultiplexedBridge_Receive_InvalidLane(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	_, err := mb.Receive(LaneType(99))
	if err == nil {
		t.Fatal("Expected error for invalid lane type")
	}
}

func TestMultiplexedBridge_GetLaneSize(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	size := mb.GetLaneSize(LaneWorkflow)
	if size != 0 {
		t.Errorf("Expected 0 messages, got %d", size)
	}

	msg := protocol.NewMessage(protocol.MessageTypeTaskRequest, []byte("test-payload"))
	mb.Send(LaneWorkflow, msg)

	size = mb.GetLaneSize(LaneWorkflow)
	if size != 1 {
		t.Errorf("Expected 1 message, got %d", size)
	}
}

func TestMultiplexedBridge_GetAllLaneSizes(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	sizes := mb.GetAllLaneSizes()
	if len(sizes) != 5 {
		t.Errorf("Expected 5 lanes, got %d", len(sizes))
	}

	// التحقق من أن جميع المسارات موجودة
	expectedLanes := []LaneType{LaneEmergency, LaneChat, LaneWorkflow, LaneFileUpload, LaneFileDownload}
	for _, laneType := range expectedLanes {
		if _, exists := sizes[laneType]; !exists {
			t.Errorf("Expected lane %s to exist", laneType.String())
		}
	}
}

func TestMultiplexedBridge_Close(t *testing.T) {
	log := logrus.New()
	mb := NewMultiplexedBridge(log)

	msg := protocol.NewMessage(protocol.MessageTypeTaskRequest, []byte("test-payload"))
	mb.Send(LaneWorkflow, msg)

	mb.Close()

	// التحقق من أن المسارات تم إغلاقها
	sizes := mb.GetAllLaneSizes()
	if len(sizes) != 0 {
		t.Errorf("Expected 0 lanes after close, got %d", len(sizes))
	}
}

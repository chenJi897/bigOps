package agent

import (
	"context"
	"errors"
	"testing"

	pb "github.com/bigops/platform/proto/gen/agent"
)

type fakeReportStream struct {
	sent   []*pb.ExecuteResponse
	closed bool
}

func (f *fakeReportStream) Send(resp *pb.ExecuteResponse) error {
	f.sent = append(f.sent, resp)
	return nil
}

func (f *fakeReportStream) CloseAndRecv() (*pb.ExecuteAck, error) {
	f.closed = true
	return &pb.ExecuteAck{Received: true}, nil
}

type fakeExecutor struct {
	panicValue any
}

func (f *fakeExecutor) Execute(ctx context.Context, task *pb.ExecuteRequest, stream reportOutputStream) {
	if f.panicValue != nil {
		panic(f.panicValue)
	}
}

func TestOpenReportStream_RetriesUntilSuccess(t *testing.T) {
	client := &AgentClient{}
	stream := &fakeReportStream{}
	attempts := 0
	client.reportOutputOpener = func() (reportOutputStream, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary error")
		}
		return stream, nil
	}

	got, err := client.openReportStream()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil {
		t.Fatalf("expected stream")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestExecuteTask_RecoversFromExecutorPanic(t *testing.T) {
	client := &AgentClient{}
	stream := &fakeReportStream{}
	client.reportOutputOpener = func() (reportOutputStream, error) {
		return stream, nil
	}

	client.executeTask(context.Background(), &fakeExecutor{panicValue: "boom"}, &pb.ExecuteRequest{
		HostResultId: 99,
		ScriptType:   "bash",
	})

	if len(stream.sent) == 0 {
		t.Fatalf("expected panic to send error message")
	}
	last := stream.sent[len(stream.sent)-1]
	if last.Phase != "error" {
		t.Fatalf("expected error phase, got %q", last.Phase)
	}
	if !stream.closed {
		t.Fatalf("expected stream to be closed after panic recovery")
	}
}

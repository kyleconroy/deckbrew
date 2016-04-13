package logtrace

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
)

type Span struct {
	sync.Mutex `json:"-"`

	// A probabilistically unique identifier for a [multi-span] trace.
	TraceID uint64

	// A probabilistically unique identifier for a span.
	SpanID uint64

	// The SpanID of this Context's parent, or 0 if there is no parent.
	ParentSpanID uint64

	// Whether the trace is sampled.
	Sampled bool `json:"-"`

	tracer ot.Tracer

	Operation string `json:"Op,omitempty"`

	// We store <start, duration> rather than <start, end> so that only
	// one of the timestamps has global clock uncertainty issues.
	Start        time.Time     `json:"Start"`
	Duration     time.Duration `json:"-"`
	Milliseconds int64         `json:"Duration"`

	// Essentially an extension mechanism. Can be used for many purposes,
	// not to be enumerated here.
	Tags ot.Tags `json:"Tags,omitempty"`

	// The span's "microlog".
	Logs []ot.LogData `json:"Logs,omitempty"`

	// The span's associated baggage.
	Baggage map[string]string `json:"Baggage,omitempty"`
}

func (s *Span) SetTag(key string, val interface{}) ot.Span {
	s.Lock()
	if s.Tags == nil {
		s.Tags = ot.Tags{}
	}
	s.Tags[key] = val
	s.Unlock()
	return s
}

func (s *Span) Finish() {
	s.FinishWithOptions(ot.FinishOptions{})
}

func (s *Span) FinishWithOptions(opts ot.FinishOptions) {
	finishTime := opts.FinishTime
	if finishTime.IsZero() {
		finishTime = time.Now().UTC()
	}
	duration := finishTime.Sub(s.Start)

	s.Lock()
	if opts.BulkLogData != nil {
		s.Logs = append(s.Logs, opts.BulkLogData...)
	}
	s.Duration = duration
	s.Milliseconds = duration.Nanoseconds() / 1000

	if blob, err := json.Marshal(s); err == nil {
		log.Println(string(blob))
	}

	s.Unlock()
}

func (s *Span) SetBaggageItem(key, val string) ot.Span {
	s.Lock()
	if s.Baggage == nil {
		s.Baggage = make(map[string]string)
	}
	s.Baggage[key] = val
	s.Unlock()
	return s
}

func (s *Span) BaggageItem(key string) string {
	s.Lock()
	val := s.Baggage[key]
	s.Unlock()
	return val
}

func (s *Span) LogEvent(event string) {
	s.Log(ot.LogData{Event: event})
}

func (s *Span) LogEventWithPayload(event string, payload interface{}) {
	s.Log(ot.LogData{Event: event, Payload: payload})
}

func (s *Span) Log(ld ot.LogData) {
	s.Lock()
	if ld.Timestamp.IsZero() {
		ld.Timestamp = time.Now().UTC()
	}
	s.Logs = append(s.Logs, ld)
	s.Unlock()
}

func (s *Span) SetOperationName(operationName string) ot.Span {
	s.Lock()
	s.Operation = operationName
	s.Unlock()
	return s
}

func (s *Span) Tracer() ot.Tracer {
	return s.tracer
}

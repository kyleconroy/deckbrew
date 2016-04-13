package logtrace

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
)

var (
	seededIDGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

func randomID() uint64 {
	seededIDLock.Lock()
	id := uint64(seededIDGen.Int63())
	seededIDLock.Unlock()
	return id
}

type Tracer struct {
}

func (t *Tracer) StartSpan(operationName string) ot.Span {
	return t.StartSpanWithOptions(
		ot.StartSpanOptions{
			OperationName: operationName,
		})
}

func (t *Tracer) StartSpanWithOptions(opts ot.StartSpanOptions) ot.Span {
	startTime := opts.StartTime
	if startTime.IsZero() {
		startTime = time.Now().UTC()
	}

	// Tags.
	tags := opts.Tags

	// Build the new span. This is the only allocation: We'll return this as
	// a opentracing.Span.
	sp := Span{}
	if opts.Parent == nil {
		sp.TraceID = randomID()
		sp.SpanID = randomID()
	} else {
		pr := opts.Parent.(*Span)
		sp.TraceID = pr.TraceID
		sp.SpanID = randomID()
		sp.ParentSpanID = pr.SpanID

		pr.Lock()
		if l := len(pr.Baggage); l > 0 {
			sp.Baggage = make(map[string]string, len(pr.Baggage))
			for k, v := range pr.Baggage {
				sp.Baggage[k] = v
			}
		}
		pr.Unlock()
	}

	sp.Operation = opts.OperationName
	sp.Start = startTime
	sp.Tags = tags
	return &sp
}

const (
	prefixTracerState     = "logtrace-"
	prefixBaggage         = "logtrace-baggage-"
	tracerStateFieldCount = 3
	fieldNameTraceID      = prefixTracerState + "traceid"
	fieldNameSpanID       = prefixTracerState + "spanid"
)

func (t *Tracer) Inject(sp ot.Span, format interface{}, carrier interface{}) error {
	switch format {
	case ot.TextMap:
		sc, ok := sp.(*Span)
		if !ok {
			return ot.ErrInvalidSpan
		}
		tmw, ok := carrier.(ot.TextMapWriter)
		if !ok {
			return ot.ErrInvalidCarrier
		}
		tmw.Set(fieldNameTraceID, strconv.FormatUint(sc.TraceID, 16))
		tmw.Set(fieldNameSpanID, strconv.FormatUint(sc.SpanID, 16))
		sc.Lock()
		for k, v := range sc.Baggage {
			tmw.Set(prefixBaggage+k, v)
		}
		sc.Unlock()
		return nil
	}
	return ot.ErrUnsupportedFormat
}

func (t *Tracer) Join(operationName string, format interface{}, carrier interface{}) (ot.Span, error) {
	switch format {
	case ot.TextMap:
	default:
		return nil, ot.ErrUnsupportedFormat
	}

	tmr, ok := carrier.(ot.TextMapReader)
	if !ok {
		return nil, ot.ErrInvalidCarrier
	}
	var traceID, propagatedSpanID uint64
	var err error
	decodedBaggage := make(map[string]string)
	err = tmr.ForeachKey(func(k, v string) error {
		switch strings.ToLower(k) {
		case fieldNameTraceID:
			traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return ot.ErrTraceCorrupted
			}
		case fieldNameSpanID:
			propagatedSpanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return ot.ErrTraceCorrupted
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t.StartSpanWithOptions(ot.StartSpanOptions{
		Parent: &Span{
			TraceID: traceID,
			SpanID:  propagatedSpanID,
			Baggage: decodedBaggage,
		},
		OperationName: operationName,
	}), nil
}

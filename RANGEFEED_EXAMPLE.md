# Rangefeed Integration for SUBSCRIBE

## How to use Rangefeed in SUBSCRIBE

### Step 1: Get Table Spans
```go
// In Subscribe() method
tableDesc := p.LookupTableByName(ctx, tableName)
tableSpan := tableDesc.PrimaryIndexSpan(p.ExecCfg().Codec)
```

### Step 2: Start Rangefeed
```go
factory := p.ExecCfg().RangeFeedFactory

eventChan := make(chan *kvpb.RangeFeedValue)

rf, err := factory.RangeFeed(
    ctx,
    "subscribe-feed",
    []roachpb.Span{tableSpan},
    hlc.Timestamp{}, // Start from now
    func(ctx context.Context, value *kvpb.RangeFeedValue) {
        // Data changed! Re-execute query
        eventChan <- value
    },
)
```

### Step 3: React to Changes
```go
// In Next() method
select {
case <-eventChan:
    // Re-execute query
    row, err := ie.QueryRowEx(ctx, ...)
    n.currentRow = row
    return true, nil
case <-time.After(1 * time.Second):
    // Timeout, return current data
    return true, nil
}
```

## Complete Flow

```
User: SUBSCRIBE TO SELECT * FROM test
         ↓
1. Parse query → Get table "test"
         ↓
2. Get table span: /Table/52/{1-2}
         ↓
3. Register rangefeed on span
         ↓
4. Execute query → Return initial results
         ↓
5. Wait for rangefeed events
         ↓
6. On event → Re-execute query
         ↓
7. Stream new results to client
         ↓
8. Repeat from step 5
```

## Key APIs

### RangeFeedFactory
```go
type Factory interface {
    RangeFeed(
        ctx context.Context,
        name string,
        spans []roachpb.Span,
        startFrom hlc.Timestamp,
        onValue func(context.Context, *kvpb.RangeFeedValue),
        opts ...Option,
    ) (*RangeFeed, error)
}
```

### RangeFeedEvent
```go
type RangeFeedValue struct {
    Key   roachpb.Key
    Value roachpb.Value
    PrevValue roachpb.Value  // If WithDiff
}
```

## Implementation Checklist

- [ ] Extract table name from query
- [ ] Get table descriptor and span
- [ ] Start rangefeed on span
- [ ] Handle rangefeed events
- [ ] Re-execute query on events
- [ ] Stream results to client
- [ ] Cleanup on disconnect

## Current Status

Currently using **polling** (every 1 second).
Need to replace with **event-driven rangefeed**.

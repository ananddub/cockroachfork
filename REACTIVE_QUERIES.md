# Reactive Query Subscriptions - Implementation Progress

## Overview
CockroachDB mein reactive query results ka implementation - queries automatically re-execute hoti hain jab underlying data change hota hai.

## Architecture
```
Client → SQL Layer (SUBSCRIBE) → Subscription Manager → Rangefeed → Storage Layer
                                        ↓
                                  Re-execute Query
                                        ↓
                                  Push Results to Client
```

## Files Modified/Created

### 1. Subscription Manager
**File**: `/pkg/sql/subscription_manager.go` ✅ CREATED

### 2. SQL AST Nodes
**File**: `/pkg/sql/sem/tree/subscribe.go` ✅ CREATED

### 3. Parser Grammar
**File**: `/pkg/sql/parser/sql.y` ✅ MODIFIED

### 4. Build Configuration
**File**: `/pkg/sql/sem/tree/BUILD.bazel` ✅ MODIFIED
- Added `subscribe.go` to source list (line 109)

### 5. SQL Execution Layer
**File**: `/pkg/sql/subscribe.go` ✅ CREATED
- `subscribeNode` - Plan node for SUBSCRIBE execution
- `unsubscribeNode` - Plan node for UNSUBSCRIBE execution
- Planner methods: `Subscribe()`, `Unsubscribe()`

## Implementation Status

### ✅ Phase 1: Core Infrastructure (COMPLETE)
- [x] Subscription manager with thread-safe tracking
- [x] AST nodes for SUBSCRIBE/UNSUBSCRIBE
- [x] Range-to-subscription mapping
- [x] Basic subscription lifecycle (subscribe/unsubscribe)

### ✅ Phase 2: Parser Integration (COMPLETE)
**File modified**: `/pkg/sql/parser/sql.y`

**Changes made**:
1. ✅ Added tokens (lines 1086-1087, 1095-1096):
   - `SUBSCRIBE` token
   - `UNSUBSCRIBE` token

2. ✅ Added to unreserved keywords (lines 18983, 19024):
   - `SUBSCRIBE` in unreserved_keyword
   - `UNSUBSCRIBE` in unreserved_keyword

3. ✅ Added to type_func_name_keyword (lines 19579, 19635):
   - `SUBSCRIBE` 
   - `UNSUBSCRIBE`

4. ✅ Added grammar rules (lines 1965-1966):
```yacc
| subscribe_stmt    // EXTEND WITH HELP: SUBSCRIBE
| unsubscribe_stmt  // EXTEND WITH HELP: UNSUBSCRIBE
```

5. ✅ Defined statement rules (lines 15468-15486):
```yacc
subscribe_stmt:
  SUBSCRIBE TO select_stmt
  {
    $$.val = &tree.Subscribe{Query: $3.slct()}
  }
| SUBSCRIBE error // SHOW HELP: SUBSCRIBE

unsubscribe_stmt:
  UNSUBSCRIBE a_expr
  {
    $$.val = &tree.Unsubscribe{SubscriptionID: $2.expr()}
  }
| UNSUBSCRIBE error // SHOW HELP: UNSUBSCRIBE
```

**Command to regenerate parser**:
```bash
./dev generate go
```

**Note**: Parser generation takes time (~5 minutes). Generated files will be in `pkg/sql/parser/`.

### ⏳ Phase 3: SQL Execution Layer (PENDING)
**Files to modify**:
- `/pkg/sql/plan.go` - Subscribe statement planning
- `/pkg/sql/conn_executor.go` - Streaming result handling
- `/pkg/sql/pgwire/conn.go` - Connection-level subscription tracking

**Implementation needed**:
1. **Query Planning**:
   - Subscribe statement ko plan karo
   - Affected ranges identify karo (optimizer se)
   - Subscription manager mein register karo

2. **Result Streaming**:
   - Initial query results send karo
   - Connection ko active rakkho
   - Updates push karo jab data change ho

3. **Connection Management**:
   - Connection close pe subscriptions cleanup
   - Subscription ID client ko return karo

### ⏳ Phase 4: Rangefeed Integration (PENDING)
**Files to modify**:
- `/pkg/sql/subscription_manager.go` - Complete `OnRangeChange()`
- `/pkg/kv/kvserver/rangefeed/` - SQL layer se connect karo

**Implementation needed**:
1. **Range Tracking**:
   - Query execution time pe ranges log karo
   - Optimizer/DistSQL se range info extract karo

2. **Change Detection**:
   - Rangefeed events ko subscription manager tak pipe karo
   - Affected subscriptions identify karo

3. **Re-execution**:
   - Query re-run karo when range changes
   - Previous results se diff calculate karo
   - Only changes client ko push karo

### ⏳ Phase 5: Protocol & Client Support (PENDING)
**Files to modify**:
- `/pkg/sql/pgwire/` - Extended protocol for streaming
- Client libraries - Subscription handling

**Options**:
1. PostgreSQL extended protocol use karo
2. WebSocket/SSE support add karo
3. Custom binary protocol

## Testing Strategy

### Unit Tests (TODO)
```bash
./dev test pkg/sql -f=TestSubscribe
./dev test pkg/sql -f=TestSubscriptionManager
```

### Integration Tests (TODO)
```bash
./dev testlogic -- --files=subscribe
```

### Test Cases Needed:
- Basic subscription lifecycle
- Multiple concurrent subscriptions
- Subscription cleanup on disconnect
- Range changes triggering re-execution
- Query with multiple ranges
- Subscription to non-existent data

## Usage Example (Future)

```sql
-- Subscribe to query
SUBSCRIBE TO SELECT * FROM users WHERE active = true;
-- Returns: subscription_id: 'abc-123-def'

-- Client receives:
-- 1. Initial results
-- 2. Updates when data changes (INSERT/UPDATE/DELETE on users table)

-- Unsubscribe
UNSUBSCRIBE 'abc-123-def';
```

## Dependencies

### Existing CockroachDB Components Used:
- **Rangefeed** (`/pkg/kv/kvserver/rangefeed/`) - Already provides change notifications
- **DistSQL** (`/pkg/sql/distsql/`) - Query execution and range tracking
- **Optimizer** (`/pkg/sql/opt/`) - Query planning and range identification
- **pgwire** (`/pkg/sql/pgwire/`) - Client protocol

### External Dependencies:
None - Pure CockroachDB implementation

## Performance Considerations

### Memory:
- Each subscription stores: query AST, range list, connection reference
- Estimated: ~1KB per subscription

### CPU:
- Re-execution overhead on every range change
- Mitigation: Debouncing, rate limiting (TODO)

### Network:
- Continuous connection per subscription
- Push only diffs, not full results (TODO)

## Known Limitations

1. **No parser integration yet** - Cannot parse SUBSCRIBE syntax
2. **No re-execution logic** - OnRangeChange() is stub
3. **No protocol support** - Cannot stream results to client
4. **No range tracking** - Query→Range mapping not implemented
5. **No diff calculation** - Sends full results, not incremental

## Next Steps

**Immediate** (Choose one):
1. Complete parser integration (manual edit of sql.y)
2. Build execution layer skeleton
3. Create end-to-end prototype without parser (for testing)

**Recommended Order**:
1. Parser integration → Test basic syntax
2. Execution layer → Test subscription lifecycle
3. Rangefeed integration → Test change detection
4. Protocol support → Test end-to-end

## References

- Rangefeed implementation: `/pkg/kv/kvserver/rangefeed/registry.go`
- CDC (similar concept): `/pkg/ccl/changefeedccl/`
- Query execution: `/pkg/sql/conn_executor.go`
- Parser grammar: `/pkg/sql/parser/sql.y`

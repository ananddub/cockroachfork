// Copyright 2025 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package sql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/security/username"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/colinfo"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sessiondata"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/uuid"
)

// Regex to extract table name
var tableNameRegex = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

func extractTableName(query string) string {
	matches := tableNameRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return ""
}

// subscribeNode represents a SUBSCRIBE statement execution.
type subscribeNode struct {
	zeroInputPlanNode
	query          tree.SelectStatement
	subscriptionID uuid.UUID
	columns        colinfo.ResultColumns

	// For query execution
	p           *planner
	currentRows []tree.Datums
	currentIdx  int
	queryString string
	tableName   string
	dbName      string
	user        username.SQLUsername

	// Event channel
	eventChan chan TableChangeEvent
}

// Subscribe implements the SUBSCRIBE statement.
func (p *planner) Subscribe(ctx context.Context, n *tree.Subscribe) (planNode, error) {
	p.curPlan.avoidBuffering = true

	queryStr := n.Query.String()
	tableName := strings.ToLower(extractTableName(queryStr))

	ie := p.ExecCfg().InternalDB.Executor()
	override := sessiondata.InternalExecutorOverride{
		User:     p.User(),
		Database: p.CurrentDatabase(),
	}

	rows, cols, err := ie.QueryBufferedExWithCols(
		ctx,
		"subscribe-init",
		nil,
		override,
		queryStr,
	)

	if err != nil || cols == nil {
		cols = colinfo.ResultColumns{{Name: "?column?", Typ: tree.DNull.ResolvedType()}}
	}

	var eventChan chan TableChangeEvent
	if p.ExecCfg().EventEmitter != nil && tableName != "" {
		eventChan = p.ExecCfg().EventEmitter.Subscribe(ctx, tableName)
	} else {
		eventChan = make(chan TableChangeEvent)
	}

	return &subscribeNode{
		query:       n.Query,
		p:           p,
		columns:     cols,
		queryString: queryStr,
		tableName:   tableName,
		dbName:      p.CurrentDatabase(),
		user:        p.User(),
		currentRows: rows,
		currentIdx:  0,
		eventChan:   eventChan,
	}, nil
}

// startExec implements the planNode interface.
func (n *subscribeNode) startExec(params runParams) error {
	n.subscriptionID = uuid.MakeV4()
	return nil
}

func (n *subscribeNode) Next(params runParams) (bool, error) {
	if n.currentIdx < len(n.currentRows) {
		n.currentIdx++
		return true, nil
	}

	for {
		select {
		case event := <-n.eventChan:
			log.VEventf(params.ctx, 1, "Event received - Table: %s, PK: %v", event.TableName, event.PrimaryKey)

			var queryToExecute string
			if event.PrimaryKey != nil && len(event.PrimaryKey) > 0 {
				var conditions []string
				for col, val := range event.PrimaryKey {
					conditions = append(conditions, "base."+col+" = "+val)
				}
				whereClause := strings.Join(conditions, " AND ")
				queryToExecute = "WITH base AS (" + n.queryString + ") SELECT * FROM base WHERE " + whereClause

				fmt.Printf("Generated query with PK filter: %s\n", queryToExecute)
			} else {
				queryToExecute = n.queryString
				fmt.Printf("No PK found, using original query\n")
			}
			fmt.Print("trigger sucessfully\n")
				
			ie := params.p.ExecCfg().InternalDB.Executor()
			override := sessiondata.InternalExecutorOverride{
				User:     n.user,
				Database: n.dbName,
			}

			rows, _, err := ie.QueryBufferedExWithCols(
				params.ctx,
				"subscribe-refresh",
				nil,
				override,
				queryToExecute,
			)

			if err != nil {
				return false, err
			}
			//TODO: fix 1 make it zero only for testing is done
			if len(rows) > 1 {

				n.currentRows = rows
				n.currentIdx = 1

				return true, nil
			}

			continue

		case <-params.ctx.Done():
			return false, params.ctx.Err()
		}
	}
}

// Values implements the planNode interface.
func (n *subscribeNode) Values() tree.Datums {
	if n.currentIdx > 0 && n.currentIdx <= len(n.currentRows) {
		return n.currentRows[n.currentIdx-1]
	}
	return tree.Datums{tree.DNull}
}

// Close implements the planNode interface.
func (n *subscribeNode) Close(ctx context.Context) {
	if n.eventChan != nil && n.tableName != "" {
		n.p.ExecCfg().EventEmitter.Unsubscribe(n.tableName, n.eventChan)
	}
}

// unsubscribeNode represents an UNSUBSCRIBE statement execution.
type unsubscribeNode struct {
	zeroInputPlanNode
	subscriptionID tree.Expr
}

func (p *planner) Unsubscribe(ctx context.Context, n *tree.Unsubscribe) (planNode, error) {
	return &unsubscribeNode{subscriptionID: n.SubscriptionID}, nil
}

func (n *unsubscribeNode) startExec(params runParams) error    { return nil }
func (n *unsubscribeNode) Next(params runParams) (bool, error) { return false, nil }
func (n *unsubscribeNode) Values() tree.Datums                 { return nil }
func (n *unsubscribeNode) Close(ctx context.Context)           {}

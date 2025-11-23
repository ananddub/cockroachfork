// Copyright 2025 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package tree

import "github.com/cockroachdb/redact"

// Subscribe represents a SUBSCRIBE TO statement for reactive queries.
type Subscribe struct {
	Query SelectStatement
}

var _ Statement = &Subscribe{}

// Format implements the NodeFormatter interface.
func (node *Subscribe) Format(ctx *FmtCtx) {
	ctx.WriteString("SUBSCRIBE TO ")
	ctx.FormatNode(node.Query)
}

// String implements the fmt.Stringer interface.
func (node *Subscribe) String() string {
	return AsString(node)
}

// StatementReturnType implements the Statement interface.
func (*Subscribe) StatementReturnType() StatementReturnType {
	return Rows
}

// StatementType implements the Statement interface.
func (*Subscribe) StatementType() StatementType {
	return TypeDML
}

// StatementTag implements the Statement interface.
func (*Subscribe) StatementTag() string {
	return "SUBSCRIBE"
}

// SafeFormat implements the redact.SafeFormatter interface.
func (node *Subscribe) SafeFormat(p redact.SafePrinter, _ rune) {
	p.SafeString("SUBSCRIBE TO ")
	p.Print(node.Query)
}

// Unsubscribe represents an UNSUBSCRIBE statement.
type Unsubscribe struct {
	SubscriptionID Expr
}

var _ Statement = &Unsubscribe{}

// Format implements the NodeFormatter interface.
func (node *Unsubscribe) Format(ctx *FmtCtx) {
	ctx.WriteString("UNSUBSCRIBE ")
	ctx.FormatNode(node.SubscriptionID)
}

// String implements the fmt.Stringer interface.
func (node *Unsubscribe) String() string {
	return AsString(node)
}

// StatementReturnType implements the Statement interface.
func (*Unsubscribe) StatementReturnType() StatementReturnType {
	return Ack
}

// StatementType implements the Statement interface.
func (*Unsubscribe) StatementType() StatementType {
	return TypeDML
}

// StatementTag implements the Statement interface.
func (*Unsubscribe) StatementTag() string {
	return "UNSUBSCRIBE"
}

// SafeFormat implements the redact.SafeFormatter interface.
func (node *Unsubscribe) SafeFormat(p redact.SafePrinter, _ rune) {
	p.SafeString("UNSUBSCRIBE ")
	p.Print(node.SubscriptionID)
}

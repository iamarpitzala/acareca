package formula

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type IService interface {
	// SyncTx replaces all formulas for a form version inside an existing transaction.
	// keyToFieldID maps field_key (e.g. "A") → field UUID resolved after field creation.
	SyncTx(ctx context.Context, tx *sqlx.Tx, formVersionID uuid.UUID, formulas []RqFormula, keyToFieldID map[string]uuid.UUID) error
	ListByFormVersionID(ctx context.Context, formVersionID uuid.UUID) ([]RsFormula, error)
}

type service struct {
	repo IRepository
}

func NewService(repo IRepository) IService {
	return &service{repo: repo}
}

func (s *service) SyncTx(ctx context.Context, tx *sqlx.Tx, formVersionID uuid.UUID, formulas []RqFormula, keyToFieldID map[string]uuid.UUID) error {
	// Full replace: delete existing then re-insert
	if err := s.repo.DeleteByFormVersionIDTx(ctx, tx, formVersionID); err != nil {
		return err
	}

	for _, rq := range formulas {
		if err := rq.Validate(); err != nil {
			return fmt.Errorf("formula %q: %w", rq.FieldKey, err)
		}

		fieldID, ok := keyToFieldID[rq.FieldKey]
		if !ok {
			return fmt.Errorf("formula references unknown field key %q", rq.FieldKey)
		}

		f := &Formula{
			ID:            uuid.New(),
			FormVersionID: formVersionID,
			FieldID:       fieldID,
			Name:          rq.Name,
		}
		if err := s.repo.CreateTx(ctx, tx, f); err != nil {
			return err
		}

		if err := insertNodes(ctx, tx, s.repo, f.ID, rq.Expression, nil, nil, keyToFieldID); err != nil {
			return fmt.Errorf("formula %q nodes: %w", rq.FieldKey, err)
		}
	}
	return nil
}

func (s *service) ListByFormVersionID(ctx context.Context, formVersionID uuid.UUID) ([]RsFormula, error) {
	formulas, err := s.repo.ListByFormVersionID(ctx, formVersionID)
	if err != nil {
		return nil, err
	}

	fieldIDToKey := map[uuid.UUID]string{}

	type formulaWithNodes struct {
		formula *Formula
		nodes   []*FormulaNodeWithKey
	}
	all := make([]formulaWithNodes, 0, len(formulas))

	for _, f := range formulas {
		nodes, err := s.repo.ListNodesWithKeyByFormulaID(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		all = append(all, formulaWithNodes{f, nodes})
		for _, n := range nodes {
			if n.FieldID != nil && n.FieldKey != nil {
				fieldIDToKey[*n.FieldID] = *n.FieldKey
			}
		}
	}

	type rsItem struct {
		rs      RsFormula
		fieldID uuid.UUID
	}
	items := make([]rsItem, 0, len(all))

	for _, fw := range all {
		fieldKey := fieldIDToKey[fw.formula.FieldID]

		rs := RsFormula{
			ID:            fw.formula.ID,
			FormVersionID: fw.formula.FormVersionID,
			FieldID:       fw.formula.FieldID,
			FieldKey:      fieldKey,
			Name:          fw.formula.Name,
			CreatedAt:     fw.formula.CreatedAt,
		}

		rootPos := int16(0)
		for _, n := range fw.nodes {
			pos := n.Position
			if n.ParentID == nil {
				pos = &rootPos
			}
			node := RsFormulaNode{
				ID:            n.ID,
				ParentID:      n.ParentID,
				NodeType:      n.NodeType,
				Operator:      n.Operator,
				FieldID:       n.FieldID,
				FieldKey:      n.FieldKey,
				ConstantValue: n.ConstantValue,
				Position:      pos,
			}
			rs.Nodes = append(rs.Nodes, node)
		}
		items = append(items, rsItem{rs, fw.formula.FieldID})
	}

	fieldIDToIdx := map[uuid.UUID]int{}
	for i, it := range items {
		fieldIDToIdx[it.fieldID] = i
	}

	n := len(items)
	deps := make([][]int, n)
	for i, fw := range all {
		for _, node := range fw.nodes {
			if node.NodeType == "FIELD" && node.FieldID != nil {
				if j, ok := fieldIDToIdx[*node.FieldID]; ok && j != i {
					deps[i] = append(deps[i], j)
				}
			}
		}
	}

	sorted := topoSort(n, deps)
	out := make([]RsFormula, 0, n)
	for _, i := range sorted {
		out = append(out, items[i].rs)
	}
	return out, nil
}

func topoSort(n int, deps [][]int) []int {
	visited := make([]bool, n)
	var order []int
	var visit func(i int)
	visit = func(i int) {
		if visited[i] {
			return
		}
		visited[i] = true
		for _, dep := range deps[i] {
			visit(dep)
		}
		order = append(order, i)
	}
	for i := 0; i < n; i++ {
		visit(i)
	}
	return order
}

// insertNodes recursively walks the expression tree and inserts rows into tbl_formula_node.
func insertNodes(ctx context.Context, tx *sqlx.Tx, repo IRepository, formulaID uuid.UUID, node *ExprNode, parentID *uuid.UUID, position *int16, keyToFieldID map[string]uuid.UUID) error {
	n := &FormulaNode{
		ID:        uuid.New(),
		FormulaID: formulaID,
		ParentID:  parentID,
		Position:  position,
	}

	switch node.Type {
	case "operator":
		n.NodeType = "OPERATOR"
		n.Operator = &node.Op
	case "field":
		n.NodeType = "FIELD"
		fid, ok := keyToFieldID[node.Key]
		if !ok {
			return fmt.Errorf("node references unknown field key %q", node.Key)
		}
		n.FieldID = &fid
	case "constant":
		n.NodeType = "CONSTANT"
		n.ConstantValue = node.Value
	}

	if err := repo.CreateNodeTx(ctx, tx, n); err != nil {
		return err
	}

	if node.Left != nil {
		pos := int16(0)
		if err := insertNodes(ctx, tx, repo, formulaID, node.Left, &n.ID, &pos, keyToFieldID); err != nil {
			return err
		}
	}
	if node.Right != nil {
		pos := int16(1)
		if err := insertNodes(ctx, tx, repo, formulaID, node.Right, &n.ID, &pos, keyToFieldID); err != nil {
			return err
		}
	}
	return nil
}

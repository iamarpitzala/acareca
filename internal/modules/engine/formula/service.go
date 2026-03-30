package formula

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type IService interface {
	SyncTx(ctx context.Context, tx *sqlx.Tx, formVersionID uuid.UUID, formulas []RqFormula, keyToFieldID map[string]uuid.UUID) error
	ListByFormVersionID(ctx context.Context, formVersionID uuid.UUID) ([]RsFormula, error)
	EvalFormulas(ctx context.Context, formVersionID uuid.UUID, keyValues map[string]float64) (map[uuid.UUID]float64, error)
}

type service struct {
	repo IRepository
}

func NewService(repo IRepository) IService {
	return &service{repo: repo}
}

func (s *service) SyncTx(ctx context.Context, tx *sqlx.Tx, formVersionID uuid.UUID, formulas []RqFormula, keyToFieldID map[string]uuid.UUID) error {
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

func (s *service) EvalFormulas(ctx context.Context, formVersionID uuid.UUID, keyValues map[string]float64) (map[uuid.UUID]float64, error) {
	formulas, err := s.ListByFormVersionID(ctx, formVersionID)
	if err != nil {
		return nil, err
	}

	vals := make(map[string]float64, len(keyValues))
	maps.Copy(vals, keyValues)

	result := make(map[uuid.UUID]float64, len(formulas))

	for _, f := range formulas {
		val, err := evalNodes(f.Nodes, vals)
		if err != nil {
			return nil, fmt.Errorf("formula %q: %w", f.Name, err)
		}
		result[f.FieldID] = val
		vals[f.FieldKey] = val
	}

	return result, nil
}

func evalNodes(nodes []RsFormulaNode, vals map[string]float64) (float64, error) {
	byID := make(map[uuid.UUID]*RsFormulaNode, len(nodes))
	for i := range nodes {
		byID[nodes[i].ID] = &nodes[i]
	}

	var root *RsFormulaNode
	for i := range nodes {
		if nodes[i].ParentID == nil {
			root = &nodes[i]
			break
		}
	}
	if root == nil {
		return 0, fmt.Errorf("formula has no root node")
	}

	return evalNode(root, byID, vals)
}

func evalNode(n *RsFormulaNode, byID map[uuid.UUID]*RsFormulaNode, vals map[string]float64) (float64, error) {
	switch n.NodeType {
	case "CONSTANT":
		if n.ConstantValue == nil {
			return 0, fmt.Errorf("constant node has nil value")
		}
		return *n.ConstantValue, nil

	case "FIELD":
		if n.FieldKey == nil {
			return 0, fmt.Errorf("field node has nil key")
		}
		v, ok := vals[*n.FieldKey]
		if !ok {
			return 0, fmt.Errorf("field key %q not found in values", *n.FieldKey)
		}
		return v, nil

	case "OPERATOR":
		if n.Operator == nil {
			return 0, fmt.Errorf("operator node has nil operator")
		}
		var left, right *RsFormulaNode
		for id, node := range byID {
			if node.ParentID != nil && *node.ParentID == n.ID {
				if node.Position != nil && *node.Position == 0 {
					cp := byID[id]
					left = cp
				} else if node.Position != nil && *node.Position == 1 {
					cp := byID[id]
					right = cp
				}
			}
		}
		if left == nil || right == nil {
			return 0, fmt.Errorf("operator %q missing children", *n.Operator)
		}
		l, err := evalNode(left, byID, vals)
		if err != nil {
			return 0, err
		}
		r, err := evalNode(right, byID, vals)
		if err != nil {
			return 0, err
		}
		switch *n.Operator {
		case "+":
			return l + r, nil
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return l / r, nil
		default:
			return 0, fmt.Errorf("unknown operator %q", *n.Operator)
		}
	}
	return 0, fmt.Errorf("unknown node type %q", n.NodeType)
}

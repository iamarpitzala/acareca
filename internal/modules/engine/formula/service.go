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

	out := make([]RsFormula, 0, len(formulas))
	for _, f := range formulas {
		nodes, err := s.repo.ListNodesByFormulaID(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		rs := RsFormula{
			ID:            f.ID,
			FormVersionID: f.FormVersionID,
			FieldID:       f.FieldID,
			Name:          f.Name,
			CreatedAt:     f.CreatedAt,
		}
		for _, n := range nodes {
			rs.Nodes = append(rs.Nodes, RsFormulaNode{
				ID:            n.ID,
				ParentID:      n.ParentID,
				NodeType:      n.NodeType,
				Operator:      n.Operator,
				FieldID:       n.FieldID,
				ConstantValue: n.ConstantValue,
				Position:      n.Position,
			})
		}
		out = append(out, rs)
	}
	return out, nil
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

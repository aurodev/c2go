// This file contains functions for transpiling binary operator expressions.

package transpiler

import (
	"github.com/elliotchance/c2go/ast"
	"github.com/elliotchance/c2go/program"
	"github.com/elliotchance/c2go/types"
	"github.com/elliotchance/c2go/util"
	goast "go/ast"
	"go/token"
	"reflect"
)

// Comma problem. Example:
// for (int i=0,j=0;i+=1,j<5;i++,j++){...}
// For solving - we have to separate the
// binary operator "," to 2 parts:
// part 1(pre ): left part  - typically one or more some expessions
// part 2(stmt): right part - always only one expression, with or witout
//               logical operators like "==", "!=", ...
func transpileBinaryOperatorComma(n *ast.BinaryOperator, p *program.Program) (
	stmt goast.Stmt, preStmts []goast.Stmt, err error) {

	left, err := transpileToStmts(n.Children[0], p)
	if err != nil {
		return nil, nil, err
	}

	right, err := transpileToStmts(n.Children[1], p)
	if err != nil {
		return nil, nil, err
	}

	preStmts = append(preStmts, left...)

	return right[0], preStmts, nil
}

func transpileBinaryOperator(n *ast.BinaryOperator, p *program.Program, exprIsStmt bool) (
	goast.Expr, string, []goast.Stmt, []goast.Stmt, error) {
	preStmts := []goast.Stmt{}
	postStmts := []goast.Stmt{}
	var err error

	operator := getTokenForOperator(n.Operator)

	// Example of C code
	// a = b = 1
	// // Operation equal transpile from right to left
	// Solving:
	// b = 1, a = b
	// // Operation comma tranpile from left to right
	// If we have for example:
	// a = b = c = 1
	// then solution is:
	// c = 1, b = c, a = b
	// |-----------|
	// this part, created in according to
	// recursive working
	// Example of AST tree for problem:
	// |-BinaryOperator 0x2f17870 <line:13:2, col:10> 'int' '='
	// | |-DeclRefExpr 0x2f177d8 <col:2> 'int' lvalue Var 0x2f176d8 'x' 'int'
	// | `-BinaryOperator 0x2f17848 <col:6, col:10> 'int' '='
	// |   |-DeclRefExpr 0x2f17800 <col:6> 'int' lvalue Var 0x2f17748 'y' 'int'
	// |   `-IntegerLiteral 0x2f17828 <col:10> 'int' 1
	//
	// Example of AST tree for solution:
	// |-BinaryOperator 0x368e8d8 <line:13:2, col:13> 'int' ','
	// | |-BinaryOperator 0x368e820 <col:2, col:6> 'int' '='
	// | | |-DeclRefExpr 0x368e7d8 <col:2> 'int' lvalue Var 0x368e748 'y' 'int'
	// | | `-IntegerLiteral 0x368e800 <col:6> 'int' 1
	// | `-BinaryOperator 0x368e8b0 <col:9, col:13> 'int' '='
	// |   |-DeclRefExpr 0x368e848 <col:9> 'int' lvalue Var 0x368e6d8 'x' 'int'
	// |   `-ImplicitCastExpr 0x368e898 <col:13> 'int' <LValueToRValue>
	// |     `-DeclRefExpr 0x368e870 <col:13> 'int' lvalue Var 0x368e748 'y' 'int'
	if getTokenForOperator(n.Operator) == token.ASSIGN {
		switch c := n.Children[1].(type) {
		case *ast.BinaryOperator:
			if getTokenForOperator(c.Operator) == token.ASSIGN {
				bSecond := ast.BinaryOperator{
					Type:     c.Type,
					Operator: "=",
				}
				bSecond.AddChild(n.Children[0])

				var impl ast.ImplicitCastExpr
				impl.Type = c.Type
				impl.Kind = "LValueToRValue"
				impl.AddChild(c.Children[0])
				bSecond.AddChild(&impl)

				var bComma ast.BinaryOperator
				bComma.Operator = ","
				bComma.Type = c.Type
				bComma.AddChild(c)
				bComma.AddChild(&bSecond)

				// exprIsStmt now changes to false to stop any AST children from
				// not being safely wrapped in a closure.
				return transpileBinaryOperator(&bComma, p, false)
			}
		}
	}

	// Example of C code
	// a = 1, b = a
	// Solving
	// a = 1; // preStmts
	// b = a; // n
	// Example of AST tree for problem:
	// |-BinaryOperator 0x368e8d8 <line:13:2, col:13> 'int' ','
	// | |-BinaryOperator 0x368e820 <col:2, col:6> 'int' '='
	// | | |-DeclRefExpr 0x368e7d8 <col:2> 'int' lvalue Var 0x368e748 'y' 'int'
	// | | `-IntegerLiteral 0x368e800 <col:6> 'int' 1
	// | `-BinaryOperator 0x368e8b0 <col:9, col:13> 'int' '='
	// |   |-DeclRefExpr 0x368e848 <col:9> 'int' lvalue Var 0x368e6d8 'x' 'int'
	// |   `-ImplicitCastExpr 0x368e898 <col:13> 'int' <LValueToRValue>
	// |     `-DeclRefExpr 0x368e870 <col:13> 'int' lvalue Var 0x368e748 'y' 'int'
	//
	// Example of AST tree for solution:
	// |-BinaryOperator 0x21a7820 <line:13:2, col:6> 'int' '='
	// | |-DeclRefExpr 0x21a77d8 <col:2> 'int' lvalue Var 0x21a7748 'y' 'int'
	// | `-IntegerLiteral 0x21a7800 <col:6> 'int' 1
	// |-BinaryOperator 0x21a78b0 <line:14:2, col:6> 'int' '='
	// | |-DeclRefExpr 0x21a7848 <col:2> 'int' lvalue Var 0x21a76d8 'x' 'int'
	// | `-ImplicitCastExpr 0x21a7898 <col:6> 'int' <LValueToRValue>
	// |   `-DeclRefExpr 0x21a7870 <col:6> 'int' lvalue Var 0x21a7748 'y' 'int'
	if getTokenForOperator(n.Operator) == token.COMMA {
		stmts, st, newPre, newPost, err := transpileToExpr(n.Children[0], p, false)
		if err != nil {
			return nil, "unknown50", nil, nil, err
		}
		preStmts = append(preStmts, newPre...)
		preStmts = append(preStmts, util.NewExprStmt(stmts))
		postStmts = append(postStmts, newPost...)
		stmts, st, newPre, newPost, err = transpileToExpr(n.Children[1], p, false)
		if err != nil {
			return nil, "unknown51", nil, nil, err
		}
		preStmts = append(preStmts, newPre...)
		postStmts = append(postStmts, newPost...)
		return stmts, st, preStmts, postStmts, nil
	}

	left, leftType, newPre, newPost, err := transpileToExpr(n.Children[0], p, false)
	if err != nil {
		return nil, "unknown52", nil, nil, err
	}

	preStmts, postStmts = combinePreAndPostStmts(preStmts, postStmts, newPre, newPost)

	right, rightType, newPre, newPost, err := transpileToExpr(n.Children[1], p, false)
	if err != nil {
		return nil, "unknown53", nil, nil, err
	}

	preStmts, postStmts = combinePreAndPostStmts(preStmts, postStmts, newPre, newPost)

	returnType := types.ResolveTypeForBinaryOperator(p, n.Operator, leftType, rightType)

	if operator == token.LAND || operator == token.LOR {
		left, err = types.CastExpr(p, left, leftType, "bool")
		p.AddMessage(ast.GenerateWarningOrErrorMessage(err, n, left == nil))
		if left == nil {
			left = util.NewNil()
		}

		right, err = types.CastExpr(p, right, rightType, "bool")
		p.AddMessage(ast.GenerateWarningOrErrorMessage(err, n, right == nil))
		if right == nil {
			right = util.NewNil()
		}

		resolvedLeftType, err := types.ResolveType(p, leftType)
		if err != nil {
			p.AddMessage(ast.GenerateWarningMessage(err, n))
		}

		expr := util.NewBinaryExpr(left, operator, right, resolvedLeftType, exprIsStmt)

		return expr, "bool", preStmts, postStmts, nil
	}

	// The right hand argument of the shift left or shift right operators
	// in Go must be unsigned integers. In C, shifting with a negative shift
	// count is undefined behaviour (so we should be able to ignore that case).
	// To handle this, cast the shift count to a uint64.
	if operator == token.SHL || operator == token.SHR {
		right, err = types.CastExpr(p, right, rightType, "unsigned long long")
		p.AddMessage(ast.GenerateWarningOrErrorMessage(err, n, right == nil))
		if right == nil {
			right = util.NewNil()
		}

		return util.NewBinaryExpr(left, operator, right, "uint64", exprIsStmt),
			leftType, preStmts, postStmts, nil
	}

	if operator == token.NEQ || operator == token.EQL {
		// Convert "(0)" to "nil" when we are dealing with equality.
		if types.IsNullExpr(right) {
			right = util.NewNil()
		} else {
			// We may have to cast the right side to the same type as the left
			// side. This is a bit crude because we should make a better
			// decision of which type to cast to instead of only using the type
			// of the left side.
			right, err = types.CastExpr(p, right, rightType, leftType)
			p.AddMessage(ast.GenerateWarningOrErrorMessage(err, n, right == nil))
		}
	}

	if operator == token.ASSIGN {
		// Memory allocation is translated into the Go-style.
		allocSize := GetAllocationSizeNode(n.Children[1])

		if allocSize != nil {
			allocSizeExpr, _, newPre, newPost, err := transpileToExpr(allocSize, p, false)
			preStmts, postStmts = combinePreAndPostStmts(preStmts, postStmts, newPre, newPost)

			if err != nil {
				return nil, "unknown60", preStmts, postStmts, err
			}

			derefType, err := types.GetDereferenceType(leftType)
			if err != nil {
				return nil, "unknown61", preStmts, postStmts, err
			}

			toType, err := types.ResolveType(p, leftType)
			if err != nil {
				return nil, "unknown62", preStmts, postStmts, err
			}

			elementSize, err := types.SizeOf(p, derefType)
			if err != nil {
				return nil, "unknown63", preStmts, postStmts, err
			}

			right = util.NewCallExpr(
				"make",
				util.NewTypeIdent(toType),
				util.NewBinaryExpr(allocSizeExpr, token.QUO, util.NewIntLit(elementSize), "int", false),
			)
		} else {
			right, err = types.CastExpr(p, right, rightType, returnType)

			if _, ok := right.(*goast.UnaryExpr); ok {
				deref, err := types.GetDereferenceType(rightType)

				if !p.AddMessage(ast.GenerateWarningMessage(err, n)) {
					resolvedDeref, err := types.ResolveType(p, deref)

					// FIXME: I'm not sure how this situation arises.
					if resolvedDeref == "" {
						resolvedDeref = "interface{}"
					}

					if !p.AddMessage(ast.GenerateWarningMessage(err, n)) {
						p.AddImport("unsafe")
						right = util.CreateSliceFromReference(resolvedDeref, right)
					}
				}
			}

			if p.AddMessage(ast.GenerateWarningMessage(err, n)) && right == nil {
				right = util.NewNil()
			}

			// Construct code for assigning value to an union field
			memberExpr, ok := n.Children[0].(*ast.MemberExpr)
			if ok {
				ref := memberExpr.GetDeclRefExpr()
				if ref != nil {
					union := p.GetStruct(ref.Type)
					if union != nil && union.IsUnion {
						attrType, err := types.ResolveType(p, ref.Type)
						if err != nil {
							p.AddMessage(ast.GenerateWarningMessage(err, memberExpr))
						}

						funcName := getFunctionNameForUnionSetter(ref.Name, attrType, memberExpr.Name)
						resExpr := util.NewCallExpr(funcName, right)
						resType := types.ResolveTypeForBinaryOperator(p, n.Operator, leftType, rightType)

						return resExpr, resType, preStmts, postStmts, nil
					}
				}
			}
		}
	}

	resolvedLeftType, err := types.ResolveType(p, leftType)
	if err != nil {
		p.AddMessage(ast.GenerateWarningMessage(err, n))
	}

	return util.NewBinaryExpr(left, operator, right, resolvedLeftType, exprIsStmt),
		types.ResolveTypeForBinaryOperator(p, n.Operator, leftType, rightType),
		preStmts, postStmts, nil
}

// GetAllocationSizeNode returns the node that, if evaluated, would return the
// size (in bytes) of a memory allocation operation. For example:
//
//     (int *)malloc(sizeof(int))
//
// Would return the node that represents the "sizeof(int)".
//
// If the node does not represent an allocation operation (such as calling
// malloc, calloc, realloc, etc.) then nil is returned.
//
// In the case of calloc() it will return a new BinaryExpr that multiplies both
// arguments.
func GetAllocationSizeNode(node ast.Node) ast.Node {
	exprs := ast.GetAllNodesOfType(node, reflect.TypeOf((*ast.CallExpr)(nil)))

	for _, expr := range exprs {
		functionName, _ := getNameOfFunctionFromCallExpr(expr.(*ast.CallExpr))

		if functionName == "malloc" {
			// Is 1 always the body in this case? Might need to be more careful
			// to find the correct node.
			return expr.(*ast.CallExpr).Children[1]
		}

		if functionName == "calloc" {
			return &ast.BinaryOperator{
				Type:     "int",
				Operator: "*",
				Children: expr.(*ast.CallExpr).Children[1:],
			}
		}

		// TODO: realloc() is not supported
		// https://github.com/elliotchance/c2go/issues/118
		//
		// Realloc will be treated as calloc which will almost certainly cause
		// bugs in your code.
		if functionName == "realloc" {
			return expr.(*ast.CallExpr).Children[2]
		}
	}

	return nil
}

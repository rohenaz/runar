package helpers

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/icellan/runar/compilers/go/codegen"
	"github.com/icellan/runar/compilers/go/frontend"
	"github.com/icellan/runar/compilers/go/ir"
)

// Artifact mirrors the compiler's Artifact type.
type Artifact struct {
	ContractName     string
	Script           string
	ASM              string
	ConstructorSlots []codegen.ConstructorSlot
	ABI              ABI
}

// ABI describes the contract's public interface.
type ABI struct {
	Methods []ABIMethod
}

// ABIMethod describes a method in the ABI.
type ABIMethod struct {
	Name     string
	IsPublic bool
}

// projectRoot returns the absolute path to the project root.
func projectRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// CompileContract compiles a .runar.ts source file with constructor args injected.
func CompileContract(sourcePath string, constructorArgs map[string]interface{}) (*Artifact, error) {
	absPath := filepath.Join(projectRoot(), sourcePath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading source: %w", err)
	}

	parseResult := frontend.ParseSource(source, absPath)
	if len(parseResult.Errors) > 0 {
		return nil, fmt.Errorf("parse errors: %v", parseResult.Errors)
	}
	if parseResult.Contract == nil {
		return nil, fmt.Errorf("no contract found in %s", sourcePath)
	}

	validResult := frontend.Validate(parseResult.Contract)
	if len(validResult.Errors) > 0 {
		return nil, fmt.Errorf("validation errors: %v", validResult.Errors)
	}

	tcResult := frontend.TypeCheck(parseResult.Contract)
	if len(tcResult.Errors) > 0 {
		return nil, fmt.Errorf("type check errors: %v", tcResult.Errors)
	}

	program := frontend.LowerToANF(parseResult.Contract)

	// Inject constructor args (must use index to modify in place)
	for i := range program.Properties {
		if val, ok := constructorArgs[program.Properties[i].Name]; ok {
			switch v := val.(type) {
			case string:
				program.Properties[i].InitialValue = v
			case float64:
				program.Properties[i].InitialValue = v
			case int64:
				program.Properties[i].InitialValue = float64(v)
			case int:
				program.Properties[i].InitialValue = float64(v)
			case *big.Int:
				program.Properties[i].InitialValue = v
			default:
				return nil, fmt.Errorf("unsupported constructor arg type for %s: %T", program.Properties[i].Name, val)
			}
		}
	}

	return compileFromProgram(program)
}

func compileFromProgram(program *ir.ANFProgram) (*Artifact, error) {
	stackMethods, err := codegen.LowerToStack(program)
	if err != nil {
		return nil, fmt.Errorf("stack lowering: %w", err)
	}

	for i := range stackMethods {
		stackMethods[i].Ops = codegen.OptimizeStackOps(stackMethods[i].Ops)
	}

	emitResult, err := codegen.Emit(stackMethods)
	if err != nil {
		return nil, fmt.Errorf("emit: %w", err)
	}

	methods := make([]ABIMethod, len(program.Methods))
	for i, m := range program.Methods {
		methods[i] = ABIMethod{Name: m.Name, IsPublic: m.IsPublic}
	}

	_ = time.Now() // suppress unused import

	return &Artifact{
		ContractName:     program.ContractName,
		Script:           emitResult.ScriptHex,
		ASM:              emitResult.ScriptAsm,
		ConstructorSlots: emitResult.ConstructorSlots,
		ABI:              ABI{Methods: methods},
	}, nil
}

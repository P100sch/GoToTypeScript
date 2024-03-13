package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"
)

const EMBEDDEDFIELD = "Embedded fields are not supported"
const NOTIMPLEMENTED = "Type is unsupported"

type typeError struct {
	text      string
	errorType types.Type
}

func (te *typeError) Error() string {
	return te.text + " : " + te.errorType.String()
}

func main() {
	if len(os.Args) == 1 || strings.ToLower(os.Args[1]) == "-h" || strings.ToLower(os.Args[1]) == "-help" {
		displayHelp(os.Args[0])
	}
	goFilePath := os.Args[1]
	tsDestinationPath := ""
	if len(os.Args) > 2 {
		tsDestinationPath = os.Args[2]
	}
	var err error
	var goFile []byte
	goFile, err = os.ReadFile(goFilePath)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			fmt.Printf("%s does not exist", goFilePath)
			os.Exit(1)
		case os.IsPermission(err):
			fmt.Printf("you have no permission to read %s", goFilePath)
			os.Exit(2)
		default:
			fmt.Println(err.Error())
			os.Exit(125)
		}
	}
	var tsDestinationFile *os.File
	if tsDestinationPath != "" {
		tsDestinationFile, err = os.Create(tsDestinationPath)
		if err != nil {
			switch {
			case os.IsExist(err):
				log.Fatalf("%s already exists", goFilePath)
			case os.IsPermission(err):
				log.Fatalf("you have no permission to write to %s", goFilePath)
			default:
				fmt.Println(err.Error())
				os.Exit(125)
			}
		}
	} else {
		tsDestinationFile = os.Stdout
	}
	err = ConvertGoFile(goFile, tsDestinationFile)
	if err != nil {
		log.Fatal(err)
	}
}

func displayHelp(scriptName string) {
	fmt.Printf("%s ./gofile.go [./destination.ts]", scriptName)
}

func ConvertGoFile(goFile []byte, tsDestinationFile *os.File) error {
	var err error
	fileSet := token.NewFileSet()
	var astFile *ast.File
	astFile, err = parser.ParseFile(fileSet, "", goFile, 0)
	if err != nil {
		return err
	}
	conf := types.Config{Importer: importer.Default()}
	var pkg *types.Package
	pkg, err = conf.Check("", fileSet, []*ast.File{astFile}, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, name := range pkg.Scope().Names() {
		object := pkg.Scope().Lookup(name)
		if named, ok := object.Type().(*types.Named); ok {
			value, err := TypeToTypeScript(named.Underlying())
			if err != nil {
				continue
			}
			tsDestinationFile.WriteString("type ")
			tsDestinationFile.WriteString(object.Name())
			tsDestinationFile.WriteString(" = ")
			tsDestinationFile.WriteString(value)
			tsDestinationFile.WriteString("\n")
		}
	}
	return nil
}

func TypeToTypeScript(t types.Type) (string, error) {
	switch v := t.(type) {
	case *types.Struct:
		return StructToTypeScript(v)
	case *types.Alias:
		return v.Obj().Name(), nil
	case *types.Array:
		value, err := TypeToTypeScript(v.Elem())
		if err != nil {
			return "", err
		}
		return "[]" + value, nil
	case *types.Basic:
		info := v.Info()
		switch {
		case info&types.IsBoolean > 0:
			return "bool", nil
		case info&types.IsComplex > 0:
			return "", &typeError{text: NOTIMPLEMENTED, errorType: v}
		case info&types.IsNumeric > 0:
			return "number", nil
		case info&types.IsString > 0:
			return "string", nil
		default:
			return "", &typeError{text: NOTIMPLEMENTED, errorType: t}
		}
	case *types.Map:
		value, err := TypeToTypeScript(v.Elem())
		if err != nil {
			return "", err
		}
		keyValue := value
		value, err = TypeToTypeScript(v.Elem())
		if err != nil {
			return "", err
		}
		return "Map<" + keyValue + ", " + value + ">", nil
	case *types.Named:
		return v.Obj().Name(), nil
	case *types.Pointer:
		value, err := TypeToTypeScript(v.Elem())
		if err != nil {
			return "", err
		}
		return "[null | " + value + "]", nil
	case *types.Slice:
		value, err := TypeToTypeScript(v.Elem())
		if err != nil {
			return "", err
		}
		return "[]" + value, nil
	case *types.Tuple:
		return TupleToTypeScript(v), nil
	case *types.Union:
		return UnionToTypeScript(v)
	default:
		return "", &typeError{text: NOTIMPLEMENTED, errorType: v}
	}
}

func StructToTypeScript(s *types.Struct) (string, error) {
	definitionBuilder := new(strings.Builder)
	definitionBuilder.WriteString("{\n")
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if f.Name() == "" {
			return "", &typeError{text: EMBEDDEDFIELD, errorType: s}
		}
		value, err := TypeToTypeScript(f.Type())
		if err != nil {
			return "", err
		}
		definitionBuilder.WriteString("  ")
		definitionBuilder.WriteString(f.Name())
		definitionBuilder.WriteString(" ")
		definitionBuilder.WriteString(value)
		definitionBuilder.WriteString("\n")
	}
	definitionBuilder.WriteString("}")
	return definitionBuilder.String(), nil
}

func TupleToTypeScript(t *types.Tuple) string {
	definitionBuilder := new(strings.Builder)
	definitionBuilder.WriteString("[")
	for i := 0; i < t.Len(); i++ {
		definitionBuilder.WriteString(t.At(i).Name())
		definitionBuilder.WriteString(", ")
	}
	return definitionBuilder.String()[0:(definitionBuilder.Len()-2)] + "]"
}

func UnionToTypeScript(u *types.Union) (string, error) {
	definitionBuilder := new(strings.Builder)
	for i := 0; i < u.Len(); i++ {
		value, err := TypeToTypeScript(u.Term(i).Type())
		if err != nil {
			return "", err
		}
		definitionBuilder.WriteString(value)
		definitionBuilder.WriteString(" | ")
	}
	return definitionBuilder.String()[0:(definitionBuilder.Len() - 3)], nil
}

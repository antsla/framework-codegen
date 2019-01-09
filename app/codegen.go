package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

func CreateHeader(out *os.File, nodeName string) {
	fmt.Fprintln(out, `package `+nodeName)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import (`)
	fmt.Fprintln(out, `	"encoding/json"`)
	fmt.Fprintln(out, `	"net/http"`)
	fmt.Fprintln(out, `	"strconv"`)
	fmt.Fprintln(out, `)`)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `const (`)
	fmt.Fprintln(out, `	errorUnauthorized  = "unauthorized"`)
	fmt.Fprintln(out, `	errorBadMethod     = "bad method"`)
	fmt.Fprintln(out, `	errorUnknownMethod = "unknown method"`)
	fmt.Fprintln(out, `)`)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `func ResponseWrite(w http.ResponseWriter, responseCode int, errorMessage string, actionResult interface{}) {`)
	fmt.Fprintln(out, `	result := make(map[string]interface{})`)
	fmt.Fprintln(out, `	result["error"] = errorMessage`)
	fmt.Fprintln(out, `	if actionResult != nil {`)
	fmt.Fprintln(out, `		result["response"] = actionResult`)
	fmt.Fprintln(out, `	}`)
	fmt.Fprintln(out, `	response, _ := json.Marshal(result)`)
	fmt.Fprintln(out, `	w.WriteHeader(responseCode)`)
	fmt.Fprintln(out, `	w.Write(response)`)
	fmt.Fprintln(out, `}`)
	fmt.Fprintln(out)
}

type Validation struct {
	Tpl string
}

type Implementation struct {
	Serve   string
	Actions []string
}

type Function struct {
	Body       string
	Variables  []string
	Validation string
}

func (function *Function) CreateValidation(fieldTag string, fieldName string, fieldType string) {
	// create map: rule => ruleValue
	attributes := make(map[string]string)
	paramname := fieldName
	for _, pair := range strings.Split(fieldTag, ",") {
		if pair == "required" {
			attributes["required"] = "required"
			continue
		}
		attribute := strings.Split(pair, "=")
		if attribute[0] == "paramname" {
			paramname = attribute[1]
			continue
		}
		attributes[attribute[0]] = attribute[1]
	}

	if fieldType == "string" {
		function.Validation += fmt.Sprintf("	%s := r.Form.Get(\"%s\")\n", fieldName, paramname)
	} else if fieldType == "int" {
		function.Validation += fmt.Sprintf("	%s, err := strconv.Atoi(r.Form.Get(\"%s\"))\n", fieldName, fieldName)
		function.Validation += fmt.Sprintf("	if err != nil {\n")
		function.Validation += fmt.Sprintf("		ResponseWrite(w, http.StatusBadRequest, \"%s must be int\", nil)\n", fieldName)
		function.Validation += fmt.Sprintf("		return\n")
		function.Validation += fmt.Sprintf("	}\n")
	}

	if defaultVal, ok := attributes["default"]; ok {
		function.Validation += fmt.Sprintf("	if %s == \"\" {\n", fieldName)
		function.Validation += fmt.Sprintf("		%s = \"%s\"\n", fieldName, defaultVal)
		function.Validation += fmt.Sprintf("	}\n")
	}
	if _, ok := attributes["required"]; ok {
		function.Validation += fmt.Sprintf("	if %s == \"\" {\n", fieldName)
		function.Validation += fmt.Sprintf("		ResponseWrite(w, http.StatusBadRequest, \"%s must be not empty\", nil)\n", fieldName)
		function.Validation += fmt.Sprintf("		return\n")
		function.Validation += fmt.Sprintf("	}\n")
	}

	for k, v := range attributes {
		switch k {
		case "min":
			lenWord := ""
			if fieldType == "string" {
				function.Validation += fmt.Sprintf("	if len(%s) < %s {\n", fieldName, v)
				lenWord = "len "
			} else if fieldType == "int" {
				function.Validation += fmt.Sprintf("	if %s < %s {\n", fieldName, v)
			}
			function.Validation += fmt.Sprintf("		ResponseWrite(w, http.StatusBadRequest, \"%s %smust be >= %s\", nil)\n", fieldName, lenWord, v)
			function.Validation += fmt.Sprintf("		return\n")
			function.Validation += fmt.Sprintf("	}\n")
		case "max":
			lenWord := ""
			if fieldType == "string" {
				function.Validation += fmt.Sprintf("	if len(%s) > %s {\n", fieldName, v)
				lenWord = "len "
			} else if fieldType == "int" {
				function.Validation += fmt.Sprintf("	if %s > %s {\n", fieldName, v)
			}
			function.Validation += fmt.Sprintf("		ResponseWrite(w, http.StatusBadRequest, \"%s %smust be <= %s\", nil)\n", fieldName, lenWord, v)
			function.Validation += fmt.Sprintf("		return\n")
			function.Validation += fmt.Sprintf("	}\n")
		case "enum":
			splitedValues := strings.Split(v, "|")
			s := make([]string, 0)
			for _, pair := range splitedValues {
				s = append(s, fmt.Sprintf("%s != \"%s\"", fieldName, pair))
			}
			function.Validation += fmt.Sprintf("	if %s {\n", strings.Join(s, " && "))
			function.Validation += fmt.Sprintf("		ResponseWrite(w, http.StatusBadRequest, \"%s must be one of [%s]\", nil)\n", fieldName, strings.Replace(v, "|", ", ", -1))
			function.Validation += fmt.Sprintf("		return\n")
			function.Validation += fmt.Sprintf("	}\n")
		}

	}
	function.Validation += fmt.Sprintf("\n")
}

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])
	CreateHeader(out, node.Name.Name)

	models := make(map[string]*Implementation)
	actions := make(map[string]*Function)

	for _, f := range node.Decls {
		switch f.(type) {
		case *ast.GenDecl: // structtures loop
			g, ok := f.(*ast.GenDecl)
			if !ok {
				fmt.Printf("[Struct] SKIP %T is not *ast.GenDecl\n", f)
				continue
			}

			for _, spec := range g.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("[Struct] SKIP %T is not *ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("[Struct] SKIP %T is not *ast.StructType\n", currType)
					continue
				}

				if len(currStruct.Fields.List) == 0 || currStruct.Fields.List[0].Tag == nil || !strings.Contains(currStruct.Fields.List[0].Tag.Value, "apivalidator:") {
					fmt.Printf("[Struct] SKIP %T is not handled structure\n", currStruct)
					continue
				}

				// creating description via structure
				actions[currType.Name.Name] = &Function{}
				for _, field := range currStruct.Fields.List {
					fieldTag := strings.Replace(field.Tag.Value, "apivalidator:", "", 1)
					fieldTag = fieldTag[2 : len(fieldTag)-2]
					fieldName := strings.ToLower(field.Names[0].Name)

					actions[currType.Name.Name].Variables = append(actions[currType.Name.Name].Variables, fieldName)
					actions[currType.Name.Name].CreateValidation(fieldTag, fieldName, field.Type.(*ast.Ident).Name)
				}
			}

		case *ast.FuncDecl: // methods loop
			g, ok := f.(*ast.FuncDecl)
			if !ok {
				fmt.Printf("[Method] SKIP %T is not *ast.GenDecl\n", f)
				continue
			}
			if g.Doc == nil {
				fmt.Printf("[Method] SKIP %#v doesnt have annotation\n", g.Name.Name)
				continue
			}

			modelName := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			if _, ok := models[modelName]; !ok { // creating new model element
				models[modelName] = &Implementation{}

				// Serve function
				models[modelName].Serve = fmt.Sprintf("// %s Model Handler\n\n", modelName)
				models[modelName].Serve += fmt.Sprintf("func (structure *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", modelName)
				models[modelName].Serve += fmt.Sprintf("	switch r.URL.Path {\n")
				models[modelName].Serve += fmt.Sprintf("{{.Actions}}")
				models[modelName].Serve += fmt.Sprintf("	default:\n")
				models[modelName].Serve += fmt.Sprintf("		ResponseWrite(w, http.StatusNotFound, errorUnknownMethod, nil)")
				models[modelName].Serve += fmt.Sprintf("	}\n")
				models[modelName].Serve += fmt.Sprintf("}\n\n")
			}

			var annotation map[string]interface{}
			json.Unmarshal([]byte(strings.Replace(g.Doc.List[0].Text, "// apigen:api ", "", 1)), &annotation)

			handler := fmt.Sprintf("	case \"%s\":\n", annotation["url"])
			if annotation["auth"] == true {
				handler += fmt.Sprintln("		if r.Header.Get(\"X-Auth\") != \"100500\" {")
				handler += fmt.Sprintln("			ResponseWrite(w, http.StatusForbidden, errorUnauthorized, nil)")
				handler += fmt.Sprintln("			return")
				handler += fmt.Sprintln("		}")
			}
			if annotation["method"] == "POST" {
				handler += fmt.Sprintln("		if r.Method != \"POST\" {")
				handler += fmt.Sprintln("			ResponseWrite(w, http.StatusNotAcceptable, errorBadMethod, nil)")
				handler += fmt.Sprintln("			return")
				handler += fmt.Sprintln("		}")
			}
			handler += fmt.Sprintf("		structure.%s%s(w, r)\n", modelName, g.Name.Name)

			models[modelName].Actions = append(models[modelName].Actions, handler)

			// Action method

			paramsModel := g.Type.Params.List[1].Type.(*ast.Ident).Name

			action := ""
			action += fmt.Sprintf("\nfunc (structure *%s) %s%s(w http.ResponseWriter, r *http.Request) {\n", modelName, modelName, g.Name.Name)
			action += fmt.Sprintf("	r.ParseForm()\n\n")
			action += fmt.Sprintf("{{.Validation}}")
			action += fmt.Sprintf("	// action\n")
			action += fmt.Sprintf("	actionResult, actionError := structure.%s(r.Context(), %s{\n", g.Name.Name, paramsModel)
			action += fmt.Sprintf("{{.Variables}}")
			action += fmt.Sprintf("	})\n\n")
			action += fmt.Sprintf("	// error handle\n")
			action += fmt.Sprintf("	if actionError != nil {\n")
			action += fmt.Sprintf("		ResponseWrite(w, actionError.HTTPStatus, actionError.Error(), nil)\n")
			action += fmt.Sprintf("		return\n")
			action += fmt.Sprintf("	}\n\n")
			action += fmt.Sprintf("	// response\n")
			action += fmt.Sprintf("	ResponseWrite(w, http.StatusOK, \"\", actionResult)\n")
			action += fmt.Sprintf("}\n")

			actions[paramsModel].Body = action
		}
	}

	// parsing map into file %s_handlers.go
	for _, implementation := range models {
		implementation.Serve = strings.Replace(implementation.Serve, "{{.Actions}}", strings.Join(implementation.Actions, ""), 1)
		fmt.Fprint(out, implementation.Serve)
	}

	for _, action := range actions {
		variables := ""
		for _, v := range action.Variables {
			variables += fmt.Sprintf("		%s: %s,\n", strings.Title(v), v)
		}
		action.Body = strings.Replace(action.Body, "{{.Validation}}", action.Validation, 1)
		action.Body = strings.Replace(action.Body, "{{.Variables}}", variables, 1)

		fmt.Fprint(out, action.Body)
	}
}

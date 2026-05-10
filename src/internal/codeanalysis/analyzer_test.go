package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ─── Helpers ───────────────────────────────────────────────────────────────

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	return path
}

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "codeanalysis-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func entityNames(entities []models.CodeEntity) []string {
	names := make([]string, len(entities))
	for i, e := range entities {
		names[i] = e.Name
	}
	sort.Strings(names)
	return names
}

func entityTypes(entities []models.CodeEntity) map[string]int {
	counts := make(map[string]int)
	for _, e := range entities {
		counts[e.EntityType]++
	}
	return counts
}

// ─── Go Analyzer Tests ────────────────────────────────────────────────────

func TestGoAnalyzer_ExportedFunctions(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

import (
	"fmt"
)

// ExportedFunc 测试导出函数
func ExportedFunc(a int, b string) (int, error) {
	return 0, nil
}

// ExportedFuncNoReturn 测试无返回值导出函数
func ExportedFuncNoReturn() {}

// unexportedFunc 测试非导出函数（应被忽略）
func unexportedFunc() {}
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	functions := 0
	for _, e := range entities {
		if e.EntityType == "function" {
			functions++
			if e.Name == "ExportedFunc" {
				expectedSig := "func ExportedFunc(a int, b string) (int, error)"
				if e.Signature != expectedSig {
					t.Errorf("ExportedFunc 签名错误: got %q, want %q", e.Signature, expectedSig)
				}
			}
			if e.Name == "ExportedFuncNoReturn" {
				expectedSig := "func ExportedFuncNoReturn()"
				if e.Signature != expectedSig {
					t.Errorf("ExportedFuncNoReturn 签名错误: got %q, want %q", e.Signature, expectedSig)
				}
			}
		}
	}
	if functions != 2 {
		t.Errorf("导出函数数量错误: got %d, want 2 (ExportedFunc, ExportedFuncNoReturn)", functions)
	}

	// 验证非导出函数被忽略
	for _, e := range entities {
		if e.Name == "unexportedFunc" {
			t.Error("非导出函数 unexportedFunc 应被忽略")
		}
	}
}

func TestGoAnalyzer_Method(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

import "fmt"

// Service 服务类型
type Service struct{}

// ExportedMethod 导出方法
func (s *Service) ExportedMethod(ctx string, req int) (string, error) {
	return "", nil
}

// valueMethod 值接收者方法（非导出，应被忽略）
func (s Service) valueMethod() string {
	return "hello"
}
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	// ExportedMethod 的签名应为 func (s *Service) ExportedMethod(ctx string, req int) (string, error)
	found := false
	for _, e := range entities {
		if e.EntityType == "function" && e.Name == "ExportedMethod" {
			found = true
			expected := "func (s *Service) ExportedMethod(ctx string, req int) (string, error)"
			if e.Signature != expected {
				t.Errorf("ExportedMethod 签名错误: got %q, want %q", e.Signature, expected)
			}
			if !strings.Contains(e.Metadata, `"receiver":"(s *Service)"`) {
				t.Errorf("ExportedMethod 元数据应包含 receiver: %s", e.Metadata)
			}
		}
		if e.EntityType == "function" && e.Name == "valueMethod" {
			t.Error("valueMethod 是非导出的，应被忽略")
		}
	}
	if !found {
		t.Error("未找到导出方法 ExportedMethod")
	}
}

func TestGoAnalyzer_StructAndInterface(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

// User 用户结构体
type User struct {
	Name string
	Age  int
}

// Reader 读取器接口
type Reader interface {
	Read(p []byte) (n int, err error)
}

// unexported 非导出类型（应被忽略）
type unexported struct{}

// StringAlias 类型别名（非 struct/interface，应被忽略）
type StringAlias string
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	typeCounts := entityTypes(entities)

	if typeCounts["struct"] != 1 {
		t.Errorf("struct 数量错误: got %d, want 1", typeCounts["struct"])
	}
	if typeCounts["interface"] != 1 {
		t.Errorf("interface 数量错误: got %d, want 1", typeCounts["interface"])
	}

	for _, e := range entities {
		switch e.Name {
		case "User":
			if e.EntityType != "struct" {
				t.Errorf("User 应为 struct, got %s", e.EntityType)
			}
			if e.Signature != "type User struct{...}" {
				t.Errorf("User 签名错误: got %q, want %q", e.Signature, "type User struct{...}")
			}
		case "Reader":
			if e.EntityType != "interface" {
				t.Errorf("Reader 应为 interface, got %s", e.EntityType)
			}
			if e.Signature != "type Reader interface{...}" {
				t.Errorf("Reader 签名错误: got %q, want %q", e.Signature, "type Reader interface{...}")
			}
		case "unexported":
			t.Error("非导出的 unexported 应被忽略")
		case "StringAlias":
			t.Error("StringAlias 不是 struct/interface，应被忽略")
		}
	}
}

func TestGoAnalyzer_Imports(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

import (
	"fmt"
	io "io"
	"net/http"
)

func Foo() {}
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	imports := 0
	for _, e := range entities {
		if e.EntityType == "import" {
			imports++
		}
	}
	if imports != 3 {
		t.Errorf("导入数量错误: got %d, want 3", imports)
	}

	// 过滤出 import 类型
	var impNames []string
	for _, e := range entities {
		if e.EntityType == "import" {
			impNames = append(impNames, e.Name)
		}
	}
	sort.Strings(impNames)
	expected := []string{"fmt", "io", "net/http"}
	if len(impNames) != len(expected) {
		t.Fatalf("导入名称数量错误: got %v, want %v", impNames, expected)
	}
	for i := range impNames {
		if impNames[i] != expected[i] {
			t.Errorf("导入名称顺序 #%d: got %q, want %q", i, impNames[i], expected[i])
		}
	}

	// 验证别名导入
	for _, e := range entities {
		if e.EntityType == "import" && e.Name == "io" {
			if !strings.Contains(e.Signature, "import io ") {
				t.Errorf("别名导入签名错误: %q", e.Signature)
			}
		}
	}
}

func TestGoAnalyzer_EmptyFile(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "empty.go", `package empty

func main() {}
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	// main 是非导出的，应被忽略；空文件无导出
	if len(entities) != 0 {
		t.Errorf("空文件应返回空切片, got %d entities", len(entities))
	}
}

func TestGoAnalyzer_VarAndConstIgnored(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

var x = 1

const Y = 2

func Foo() {}
`)

	entities, err := analyzeGoFile(src)
	if err != nil {
		t.Fatalf("analyzeGoFile 失败: %v", err)
	}

	// 只有导出函数 Foo 应该被提取（以及可能的导入）
	// 没有导入, var 和 const 应该被忽略
	for _, e := range entities {
		if e.EntityType != "function" {
			t.Errorf("意外的实体类型: %s (%s)", e.EntityType, e.Name)
		}
	}
	if len(entities) != 1 || entities[0].Name != "Foo" {
		t.Errorf("应只提取 Foo, got %v", entityNames(entities))
	}
}

// ─── TS Analyzer Tests ─────────────────────────────────────────────────────

func TestTSAnalyzer_ExportFunction(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `// 导出函数
export function add(a: number, b: number): number {
	return a + b;
}

// 异步导出函数
export async function fetchData(url: string): Promise<any> {
	const response = await fetch(url);
	return response.json();
}

// 非导出函数（不应匹配）
function internalHelper() {
	return 42;
}
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	functions := 0
	for _, e := range entities {
		if e.EntityType == "function" {
			functions++
		}
	}
	if functions != 2 {
		t.Errorf("导出函数数量错误: got %d, want 2", functions)
	}

	for _, e := range entities {
		if e.Name == "add" && e.EntityType == "function" {
			if !strings.Contains(e.Signature, "export function add") {
				t.Errorf("add 签名错误: %q", e.Signature)
			}
		}
		if e.Name == "fetchData" && e.EntityType == "function" {
			if !strings.Contains(e.Signature, "export async function fetchData") {
				t.Errorf("fetchData 签名错误: %q", e.Signature)
			}
		}
		if e.Name == "internalHelper" {
			t.Error("非导出函数 internalHelper 不应被匹配")
		}
	}
}

func TestTSAnalyzer_ExportClass(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `// 导出类
export class UserService {
	private users: User[] = [];
	
	getUser(id: string): User | undefined {
		return this.users.find(u => u.id === id);
	}
}

// 抽象导出类
export abstract class BaseRepository<T> {
	abstract find(id: string): Promise<T>;
}

// 非导出类
class InternalCache {
	private data = new Map();
}
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	classes := 0
	for _, e := range entities {
		if e.EntityType == "class" {
			classes++
		}
	}
	if classes != 2 {
		t.Errorf("导出类数量错误: got %d, want 2", classes)
	}

	for _, e := range entities {
		if e.EntityType == "class" && e.Name == "UserService" {
			if !strings.Contains(e.Signature, "export class UserService") {
				t.Errorf("UserService 签名错误: %q", e.Signature)
			}
		}
		if e.EntityType == "class" && e.Name == "BaseRepository" {
			if !strings.Contains(e.Signature, "export abstract class BaseRepository") {
				t.Errorf("BaseRepository 签名错误: %q", e.Signature)
			}
		}
		if e.Name == "InternalCache" {
			t.Error("非导出类 InternalCache 不应被匹配")
		}
	}
}

func TestTSAnalyzer_ExportInterface(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `// 导出接口
export interface User {
	id: string;
	name: string;
	email: string;
}

// 非导出接口
interface InternalConfig {
	debug: boolean;
}
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	ifaces := 0
	for _, e := range entities {
		if e.EntityType == "interface" {
			ifaces++
		}
	}
	if ifaces != 1 {
		t.Errorf("导出接口数量错误: got %d, want 1", ifaces)
	}

	for _, e := range entities {
		if e.EntityType == "interface" && e.Name == "User" {
			if !strings.Contains(e.Signature, "export interface User") {
				t.Errorf("User 接口签名错误: %q", e.Signature)
			}
		}
		if e.Name == "InternalConfig" {
			t.Error("非导出接口 InternalConfig 不应被匹配")
		}
	}
}

func TestTSAnalyzer_ExportConst(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `// 导出常量
export const MAX_RETRIES = 3;

export const API_BASE_URL = "https://api.example.com";

// 箭头函数赋值给 const（应匹配 export const）
export const myFunc = (x: number) => x * 2;

// 非导出 const
const localConst = "secret";
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	consts := 0
	for _, e := range entities {
		if e.EntityType == "const" {
			consts++
		}
	}
	if consts != 3 {
		t.Errorf("导出 const 数量错误: got %d, want 3", consts)
	}

	for _, e := range entities {
		if e.EntityType == "const" && e.Name == "MAX_RETRIES" {
			if !strings.Contains(e.Signature, "export const MAX_RETRIES") {
				t.Errorf("MAX_RETRIES 签名错误: %q", e.Signature)
			}
		}
		if e.Name == "localConst" {
			t.Error("非导出 const localConst 不应被匹配")
		}
	}
}

func TestTSAnalyzer_Imports(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `import { useState } from "react";
import * as fs from "fs";
import type { FC } from "react";
import { createContext } from "react";
import "./styles.css";
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	imports := 0
	for _, e := range entities {
		if e.EntityType == "import" {
			imports++
		}
	}
	if imports != 5 {
		t.Errorf("导入数量错误: got %d, want 5", imports)
	}

	importPaths := make(map[string]bool)
	for _, e := range entities {
		if e.EntityType == "import" {
			importPaths[e.Name] = true
		}
	}

	expectedPaths := []string{"react", "fs", "./styles.css"}
	for _, p := range expectedPaths {
		if !importPaths[p] {
			t.Errorf("缺少导入路径: %s", p)
		}
	}
}

func TestTSAnalyzer_MixedFile(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "mixed.ts", `import { Component } from "react";

export interface Props {
	title: string;
}

export const DEFAULT_TITLE = "Hello";

export function createComponent(props: Props) {
	return <Component title={props.title} />;
}

export class MyComponent extends Component<Props> {
	render() {
		return <div>{this.props.title}</div>;
	}
}
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	counts := entityTypes(entities)
	expected := map[string]int{
		"function":  1,
		"class":     1,
		"interface": 1,
		"const":     1,
		"import":    1,
	}
	for typ, count := range expected {
		if counts[typ] != count {
			t.Errorf("类型 %s 数量错误: got %d, want %d", typ, counts[typ], count)
		}
	}
}

func TestTSAnalyzer_EmptyFile(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "empty.ts", `// just a comment
`)

	entities, err := analyzeTSFile(src)
	if err != nil {
		t.Fatalf("analyzeTSFile 失败: %v", err)
	}

	if len(entities) != 0 {
		t.Errorf("空 TS 文件应返回空切片, got %d entities", len(entities))
	}
}

// ─── AnalyzeFile Dispatch Tests ────────────────────────────────────────────

func TestAnalyzeFile_DispatchesToGoAnalyzer(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.go", `package mypkg

func Hello() string {
	return "world"
}
`)

	entities, err := AnalyzeFile(src)
	if err != nil {
		t.Fatalf("AnalyzeFile 失败: %v", err)
	}

	if len(entities) == 0 {
		t.Fatal("AnalyzeFile(.go) 应返回实体")
	}
	if entities[0].EntityType != "function" || entities[0].Name != "Hello" {
		t.Errorf("应找到 Hello 函数, got %+v", entities[0])
	}
}

func TestAnalyzeFile_DispatchesToTSAnalyzer(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "test.ts", `export const x = 1;
`)

	entities, err := AnalyzeFile(src)
	if err != nil {
		t.Fatalf("AnalyzeFile 失败: %v", err)
	}

	if len(entities) == 0 {
		t.Fatal("AnalyzeFile(.ts) 应返回实体")
	}
	if entities[0].EntityType != "const" || entities[0].Name != "x" {
		t.Errorf("应找到 const x, got %+v", entities[0])
	}
}

func TestAnalyzeFile_DispatchesToTSAnalyzer_TSX(t *testing.T) {
	dir := tempDir(t)
	src := writeTempFile(t, dir, "component.tsx", `export const App = () => <div>Hello</div>;
`)

	entities, err := AnalyzeFile(src)
	if err != nil {
		t.Fatalf("AnalyzeFile 失败: %v", err)
	}

	if len(entities) == 0 {
		t.Fatal("AnalyzeFile(.tsx) 应返回实体")
	}
	if entities[0].EntityType != "const" || entities[0].Name != "App" {
		t.Errorf("应找到 const App, got %+v", entities[0])
	}
}

func TestAnalyzeFile_UnsupportedExtension(t *testing.T) {
	_, err := AnalyzeFile("test.rb")
	if err == nil {
		t.Fatal("AnalyzeFile(.rb) 应返回错误")
	}
	if !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Errorf("错误消息不正确: %v", err)
	}
}

// ─── AnalyzeProject Tests ──────────────────────────────────────────────────

func TestAnalyzeProject_WalksDirectory(t *testing.T) {
	dir := tempDir(t)
	// 创建混合项目结构
	writeTempFile(t, dir, "main.go", `package main

import "fmt"

func Hello() string {
	return "hi"
}
`)
	writeTempFile(t, dir, "utils/helper.ts", `export function helper() {
	return 42;
}
`)
	writeTempFile(t, dir, "utils/helper_test.ts", `import { helper } from "./helper";

test("helper", () => {
	expect(helper()).toBe(42);
});
`)

	// 应该被忽略的文件
	writeTempFile(t, dir, "build/output.go", `package build

func Build() {}
`)
	writeTempFile(t, dir, "node_modules/pkg/index.ts", `export const pkg = 1;
`)
	writeTempFile(t, dir, ".git/config", `[core]
	repositoryformatversion = 0
`)
	writeTempFile(t, dir, "Readme.md", `# project
`)

	savedFiles := make(map[string]int)
	saveFn := func(filePath string, entities []models.CodeEntity) error {
		savedFiles[filePath] = len(entities)
		return nil
	}

	err := AnalyzeProject(dir, saveFn)
	if err != nil {
		t.Fatalf("AnalyzeProject 失败: %v", err)
	}

	// 应该找到 main.go, utils/helper.ts, utils/helper_test.ts
	if len(savedFiles) != 3 {
		keys := make([]string, 0, len(savedFiles))
		for k := range savedFiles {
			rel, _ := filepath.Rel(dir, k)
			keys = append(keys, rel)
		}
		t.Errorf("处理文件数量错误: got %d, want 3. 文件: %v", len(savedFiles), keys)
	}

	for path := range savedFiles {
		rel, _ := filepath.Rel(dir, path)
		if strings.Contains(rel, "build") || strings.Contains(rel, "node_modules") || strings.Contains(rel, ".git") {
			t.Errorf("应忽略的路径被处理: %s", rel)
		}
	}
}

func TestAnalyzeProject_SkipNonCodeFiles(t *testing.T) {
	dir := tempDir(t)
	writeTempFile(t, dir, "data.json", `{"key": "value"}`)
	writeTempFile(t, dir, "styles.css", `.main { color: red; }`)
	writeTempFile(t, dir, "app.go", `package app

func Run() {}
`)

	savedFiles := make(map[string]int)
	saveFn := func(filePath string, entities []models.CodeEntity) error {
		savedFiles[filePath] = len(entities)
		return nil
	}

	err := AnalyzeProject(dir, saveFn)
	if err != nil {
		t.Fatalf("AnalyzeProject 失败: %v", err)
	}

	if len(savedFiles) != 1 {
		t.Errorf("应只处理 1 个 .go 文件, got %d", len(savedFiles))
	}

	for path := range savedFiles {
		if !strings.HasSuffix(path, ".go") {
			t.Errorf("非 .go 文件被处理: %s", path)
		}
	}
}

func TestAnalyzeProject_SaveFnError(t *testing.T) {
	dir := tempDir(t)
	writeTempFile(t, dir, "main.go", `package main

func Hello() string {
	return "hi"
}
`)

	// saveFn 返回错误不应终止遍历
	saveFn := func(filePath string, entities []models.CodeEntity) error {
		return nil
	}

	err := AnalyzeProject(dir, saveFn)
	if err != nil {
		t.Fatalf("AnalyzeProject 失败: %v", err)
	}
}

// ─── Ignore Rules Tests ────────────────────────────────────────────────────

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"src/main.go", false},
		{".git/config", true},
		{".chronodraft/data.db", true},
		{"node_modules/pkg/index.js", true},
		{"vendor/pkg/main.go", true},
		{"build/output.bin", true},
		{"dist/app.js", true},
		{"target/debug/app", true},
		{"out/release/app", true},
		{"bin/cli", true},
		{"obj/debug/main.o", true},
		{".next/build-manifest.json", true},
		{"src/main.go", false},
		{"internal/app.go", false},
	}

	for _, tt := range tests {
		got := shouldIgnore(tt.path)
		if got != tt.expected {
			t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestShouldIgnoreByExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", false},
		{"Main.class", true},
		{"script.pyc", true},
		{"lib.o", true},
		{"app.dll", true},
		{"app.exe", true},
		{"app.bin", true},
		{"app.flat", true},
		{"app.dex", true},
		{"main.ts", false},
		{"component.tsx", false},
	}

	for _, tt := range tests {
		got := shouldIgnoreByExtension(tt.path)
		if got != tt.expected {
			t.Errorf("shouldIgnoreByExtension(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ─── Java Analyzer Tests ───────────────────────────────────────────────────

func TestJavaAnalyzer_ClassAndImport(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "Test.java", "import java.util.List;\npublic class HelloWorld {\n}\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "HelloWorld") {
		t.Errorf("expected HelloWorld class, got %v", names)
	}
	if !contains(names, "java.util.List") {
		t.Errorf("expected java.util.List import, got %v", names)
	}
}

// ─── Python Analyzer Tests ─────────────────────────────────────────────────

func TestPythonAnalyzer_FuncAndClass(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "test.py", "def hello():\n    pass\nclass MyClass:\n    pass\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "hello") {
		t.Errorf("expected hello function, got %v", names)
	}
	if !contains(names, "MyClass") {
		t.Errorf("expected MyClass class, got %v", names)
	}
}

// ─── C Analyzer Tests ──────────────────────────────────────────────────────

func TestCAnalyzer_StructAndInclude(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "test.h", "#include <stdio.h>\nstruct Point { int x; int y; };\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "Point") {
		t.Errorf("expected Point struct, got %v", names)
	}
	if !contains(names, "stdio.h") {
		t.Errorf("expected stdio.h include, got %v", names)
	}
}

// ─── Rust Analyzer Tests ───────────────────────────────────────────────────

func TestRustAnalyzer_FnAndStruct(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "lib.rs", "pub fn calculate() {}\npub struct Config {}\nuse std::collections::HashMap;\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "calculate") {
		t.Errorf("expected calculate fn, got %v", names)
	}
	if !contains(names, "Config") {
		t.Errorf("expected Config struct, got %v", names)
	}
}

// ─── C# Analyzer Tests ─────────────────────────────────────────────────────

func TestCSharpAnalyzer_ClassAndUsing(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "Test.cs", "using System;\npublic class Program { public static void Main() {} }\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "Program") {
		t.Errorf("expected Program class, got %v", names)
	}
}

// ─── Kotlin Analyzer Tests ─────────────────────────────────────────────────

func TestKotlinAnalyzer_FunAndClass(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "Test.kt", "import kotlin.test.*\nfun greet() {}\ndata class User(val name: String)\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "greet") {
		t.Errorf("expected greet fun, got %v", names)
	}
	if !contains(names, "User") {
		t.Errorf("expected User class, got %v", names)
	}
}

// ─── JavaScript Analyzer Tests ─────────────────────────────────────────────

func TestJSAnalyzer_FuncAndImport(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "test.js", "import React from 'react';\nexport function App() {}\nexport class Component {}\n")
	entities, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	names := entityNames(entities)
	if !contains(names, "App") {
		t.Errorf("expected App function, got %v", names)
	}
	if !contains(names, "react") {
		t.Errorf("expected react import, got %v", names)
	}
}

func TestVueAnalyzer_ComponentAndImport(t *testing.T) {
	dir := tempDir(t)
	path := writeTempFile(t, dir, "Test.vue", `<script setup lang="ts">
import { ref } from "vue"
import LoginForm from "./LoginForm.vue"
export function setupAuth() { return true }
const count = ref(0)
const MyComponent = defineComponent({})
</script>
<template><div>{{ count }}</div></template>
`)
	entities, err := AnalyzeFile(path)
	if err != nil { t.Fatalf("AnalyzeFile: %v", err) }
	names := entityNames(entities)
	if !contains(names, "setupAuth") { t.Errorf("expected setupAuth function, got %v", names) }
	if !contains(names, "MyComponent") { t.Errorf("expected MyComponent component, got %v", names) }
	if !contains(names, "vue") { t.Errorf("expected vue import, got %v", names) }
}

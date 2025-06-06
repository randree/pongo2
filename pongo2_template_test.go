package pongo2_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/randree/pongo2/v7"
)

type stringerValueType int

func (v stringerValueType) String() string {
	return "-" + strconv.Itoa(int(v)) + ":"
}

var (
	strPtr    = stringerValueType(1234)
	adminList = []string{"user2"}
)

var (
	time1 = time.Date(2014, 0o6, 10, 15, 30, 15, 0, time.UTC)
	time2 = time.Date(2011, 0o3, 21, 8, 37, 56, 12, time.UTC)
)

type post struct {
	Text    string
	Created time.Time
}

type user struct {
	Name      string
	Validated bool
}

type comment struct {
	Author *user
	Date   time.Time
	Text   string
}

func isAdmin(u *user) bool {
	for _, a := range adminList {
		if a == u.Name {
			return true
		}
	}
	return false
}

func (u *user) Is_admin() *pongo2.Value {
	return pongo2.AsValue(isAdmin(u))
}

func (u *user) Is_admin2() bool {
	return isAdmin(u)
}

func (p *post) String() string {
	return ":-)"
}

/*
 * Start setup sandbox
 */

type tagSandboxDemoTag struct{}

func (node *tagSandboxDemoTag) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	writer.WriteString("hello")
	return nil
}

func tagSandboxDemoTagParser(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	return &tagSandboxDemoTag{}, nil
}

func BannedFilterFn(in *pongo2.Value, params *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return in, nil
}

func init() {
	pongo2.DefaultSet.Debug = true

	pongo2.RegisterFilter("banned_filter", BannedFilterFn)
	pongo2.RegisterFilter("unbanned_filter", BannedFilterFn)
	pongo2.RegisterTag("banned_tag", tagSandboxDemoTagParser)
	pongo2.RegisterTag("unbanned_tag", tagSandboxDemoTagParser)

	pongo2.DefaultSet.BanFilter("banned_filter")
	pongo2.DefaultSet.BanTag("banned_tag")

	f, err := os.CreateTemp(os.TempDir(), "pongo2_")
	if err != nil {
		panic(fmt.Sprintf("cannot write to %s", os.TempDir()))
	}
	defer f.Close()
	_, err = f.Write([]byte("Hello from pongo2"))
	if err != nil {
		panic(fmt.Sprintf("cannot write to %s", os.TempDir()))
	}
	pongo2.DefaultSet.Globals["temp_file"] = f.Name()
}

/*
 * End setup sandbox
 */

var tplContext = pongo2.Context{
	"number": 11,
	"simple": map[string]any{
		"number":                   42,
		"name":                     "john doe",
		"included_file":            "INCLUDES.helper",
		"included_file_not_exists": "INCLUDES.helper.not_exists",
		"nil":                      nil,
		"uint":                     uint(8),
		"float":                    float64(3.1415),
		"str":                      "string",
		"chinese_hello_world":      "你好世界",
		"bool_true":                true,
		"bool_false":               false,
		"newline_text": `this is a text
with a new line in it`,
		"long_text": `This is a simple text.

This too, as a paragraph.
Right?

Yep!`,
		"escape_js_test":     `escape sequences \r\n\'\" special chars "?!=$<>`,
		"one_item_list":      []int{99},
		"multiple_item_list": []int{1, 1, 2, 3, 5, 8, 13, 21, 34, 55},
		"unsorted_int_list":  []int{192, 581, 22, 1, 249, 9999, 1828591, 8271},
		"fixed_item_list":    [...]int{1, 2, 3, 4},
		"misc_list":          []any{"Hello", 99, 3.14, "good"},
		"escape_text":        "This is \\a Test. \"Yep\". 'Yep'.",
		"xss":                "<script>alert(\"uh oh\");</script>",
		"time1":              time1,
		"time2":              time2,
		"stringer":           strPtr,
		"stringerPtr":        &strPtr,
		"intmap": map[int]string{
			1: "one",
			5: "five",
			2: "two",
		},
		"strmap": map[string]string{
			"abc": "def",
			"bcd": "efg",
			"zab": "cde",
			"gh":  "kqm",
			"ukq": "qqa",
			"aab": "aba",
		},
		"func_add": func(a, b int) int {
			return a + b
		},
		"func_add_iface": func(a, b any) any {
			x, is1 := a.(int)
			y, is2 := b.(int)
			if is1 && is2 {
				return x + y
			}
			return 0
		},
		"func_variadic": func(msg string, args ...any) string {
			return fmt.Sprintf(msg, args...)
		},
		"func_variadic_sum_int": func(args ...int) int {
			// Create a sum
			s := 0
			for _, i := range args {
				s += i
			}
			return s
		},
		"func_variadic_sum_int2": func(args ...*pongo2.Value) *pongo2.Value {
			// Create a sum
			s := 0
			for _, i := range args {
				s += i.Integer()
			}
			return pongo2.AsValue(s)
		},
		"func_ensure_nil": func(x any) bool {
			return x == nil
		},
		"func_ensure_nil_variadic": func(args ...any) bool {
			for _, i := range args {
				if i != nil {
					return false
				}
			}
			return true
		},
	},
	"complex": map[string]any{
		"is_admin": isAdmin,
		"post": post{
			Text:    "<h2>Hello!</h2><p>Welcome to my new blog page. I'm using pongo2 which supports {{ variables }} and {% tags %}.</p>",
			Created: time2,
		},
		"comments": []*comment{
			{
				Author: &user{
					Name:      "user1",
					Validated: true,
				},
				Date: time1,
				Text: "\"pongo2 is nice!\"",
			},
			{
				Author: &user{
					Name:      "user2",
					Validated: true,
				},
				Date: time2,
				Text: "comment2 with <script>unsafe</script> tags in it",
			},
			{
				Author: &user{
					Name:      "user3",
					Validated: false,
				},
				Date: time1,
				Text: "<b>hello!</b> there",
			},
		},
		"comments2": []*comment{
			{
				Author: &user{
					Name:      "user1",
					Validated: true,
				},
				Date: time2,
				Text: "\"pongo2 is nice!\"",
			},
			{
				Author: &user{
					Name:      "user1",
					Validated: true,
				},
				Date: time1,
				Text: "comment2 with <script>unsafe</script> tags in it",
			},
			{
				Author: &user{
					Name:      "user3",
					Validated: false,
				},
				Date: time1,
				Text: "<b>hello!</b> there",
			},
		},
	},
}

func TestTemplate_Functions(t *testing.T) {
	mydict := map[string]any{
		"foo":    "bar",
		"foobar": 8379,
	}

	tests := []struct {
		name         string
		template     string
		context      pongo2.Context
		want         string
		errorMessage string
		wantErr      bool
	}{
		{
			name:     "NoError",
			template: "{{ testFunc(mydict) }}",
			context: pongo2.Context{
				"mydict": mydict,
				"testFunc": func(i any) (string, error) {
					d, err := json.Marshal(i)
					return string(d), err
				},
			},
			want:    `{&quot;foo&quot;:&quot;bar&quot;,&quot;foobar&quot;:8379}`,
			wantErr: false,
		},
		{
			name:     "WithError",
			template: "{{ testFunc(mydict) }}",
			context: pongo2.Context{
				"mydict": mydict,
				"testFunc": func(i any) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			errorMessage: "[Error (where: execution) in <string> | Line 1 Col 4 near 'testFunc'] something went wrong",
			wantErr:      true,
		},
		{
			name:     "TooMuchArguments",
			template: "{{ testFunc(mydict) }}",
			context: pongo2.Context{
				"mydict": mydict,
				"testFunc": func(i any) (string, int, error) {
					return "", 0, nil
				},
			},
			errorMessage: "[Error (where: execution) in <string> | Line 1 Col 4 near 'testFunc'] 'testFunc' must have exactly 1 or 2 output arguments, the second argument must be of type error",
			wantErr:      true,
		},
		{
			name:     "InvalidArguments",
			template: "{{ testFunc(mydict) }}",
			context: pongo2.Context{
				"mydict": map[string]any{
					"foo":    "bar",
					"foobar": 8379,
				},
				"testFunc": func(i any) (string, int) {
					return "", 0
				},
			},
			errorMessage: "[Error (where: execution) in <string> | Line 1 Col 4 near 'testFunc'] the second return value is not an error",
			wantErr:      true,
		},
		{
			name:     "NilToNonNilParameter",
			template: "{{ testFunc(nil) }}",
			context: pongo2.Context{
				"testFunc": func(i int) int {
					return 1
				},
			},
			errorMessage: "[Error (where: execution) in <string> | Line 1 Col 4 near 'testFunc'] function input argument 0 of 'testFunc' must be of type int or *pongo2.Value (not <nil>)",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl, _ := pongo2.FromString("{{ testFunc(mydict) }}")
			got, err := tpl.Execute(tt.context)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Template.Execute() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.errorMessage {
					t.Errorf("Template.Execute() error = %v, expected error %v", err, tt.errorMessage)
					return
				}
			}
			if got != tt.want {
				t.Errorf("Template.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplates(t *testing.T) {
	// Add a global to the default set
	pongo2.Globals["this_is_a_global_variable"] = "this is a global text"

	matches, err := filepath.Glob("./template_tests/*.tpl")
	if err != nil {
		t.Fatal(err)
	}
	for idx, match := range matches {
		t.Run(fmt.Sprintf("%03d-%s", idx+1, match), func(t *testing.T) {
			t.Logf("[Template %3d] Testing '%s'", idx+1, match)
			tpl, err := pongo2.FromFile(match)
			if err != nil {
				t.Fatalf("Error on FromFile('%s'): %s", match, err.Error())
			}

			// Read options from file
			optsStr, _ := os.ReadFile(fmt.Sprintf("%s.options", match))
			trimBlocks := strings.Contains(string(optsStr), "TrimBlocks=true")
			lStripBlocks := strings.Contains(string(optsStr), "LStripBlocks=true")

			tpl.Options.TrimBlocks = trimBlocks
			tpl.Options.LStripBlocks = lStripBlocks

			testFilename := fmt.Sprintf("%s.out", match)
			testOut, rerr := os.ReadFile(testFilename)
			if rerr != nil {
				t.Fatalf("Error on ReadFile('%s'): %s", testFilename, rerr.Error())
			}
			tplOut, err := tpl.ExecuteBytes(tplContext)
			if err != nil {
				t.Fatalf("Error on Execute('%s'): %s", match, err.Error())
			}
			tplOut = testTemplateFixes.fixIfNeeded(match, tplOut)
			if !bytes.Equal(testOut, tplOut) {
				t.Logf("Template (rendered) '%s': '%s'", match, tplOut)
				errFilename := filepath.Base(fmt.Sprintf("%s.error", match))
				err := os.WriteFile(errFilename, []byte(tplOut), 0o600)
				if err != nil {
					t.Fatalf(err.Error())
				}
				t.Logf("get a complete diff with command: 'diff -ya %s %s'", testFilename, errFilename)
				t.Errorf("Failed: test_out != tpl_out for %s", match)
			}
		})
	}
}

func TestBlockTemplates(t *testing.T) {
	// debug = true

	matches, err := filepath.Glob("./template_tests/block_render/*.tpl")
	if err != nil {
		t.Fatal(err)
	}
	for idx, match := range matches {
		t.Run(fmt.Sprintf("%03d-%s", idx+1, match), func(t *testing.T) {
			t.Logf("[BlockTemplate %3d] Testing '%s'", idx+1, match)

			tpl, err := pongo2.FromFile(match)
			if err != nil {
				t.Fatalf("Error on FromFile('%s'): %s", match, err.Error())
			}

			testFilename := fmt.Sprintf("%s.out", match)
			testOut, rerr := os.ReadFile(testFilename)
			if rerr != nil {
				t.Fatalf("Error on ReadFile('%s'): %s", testFilename, rerr.Error())
			}
			tpl_out, err := tpl.ExecuteBlocks(tplContext, []string{"content", "more_content"})
			if err != nil {
				t.Fatalf("Error on ExecuteBlocks('%s'): %s", match, err.Error())
			}

			if _, ok := tpl_out["content"]; !ok {
				t.Errorf("Failed: content not in tpl_out for %s", match)
			}
			if _, ok := tpl_out["more_content"]; !ok {
				t.Errorf("Failed: more_content not in tpl_out for %s", match)
			}
			testString := string(testOut[:])
			joinedString := strings.Join([]string{tpl_out["content"], tpl_out["more_content"]}, "")
			if testString != joinedString {
				t.Logf("BlockTemplate (rendered) '%s': '%s'", match, tpl_out["content"])
				errFilename := filepath.Base(fmt.Sprintf("%s.error", match))
				err := os.WriteFile(errFilename, []byte(joinedString), 0o600)
				if err != nil {
					t.Fatalf(err.Error())
				}
				t.Logf("get a complete diff with command: 'diff -ya %s %s'", testFilename, errFilename)
				t.Errorf("Failed: test_out != tpl_out for %s", match)
			}
		})
	}
}

type testTemplateFixesT map[*regexp.Regexp]func(string) string

func (instance testTemplateFixesT) fixIfNeeded(name string, in []byte) []byte {
	out := string(in)
	for r, f := range instance {
		if r.MatchString(name) {
			out = f(out)
		}
	}
	return []byte(out)
}

var testTemplateFixes = testTemplateFixesT{
	regexp.MustCompile(`.*template_tests[/\\]macro\.tpl`): func(in string) string {
		out := regexp.MustCompile(`(?:\.[/\\]|)(template_tests)[/\\](macro\.tpl)`).ReplaceAllString(in, "$1/$2")
		return out
	},
}

func TestExecutionErrors(t *testing.T) {
	// debug = true

	matches, err := filepath.Glob("./template_tests/*-execution.err")
	if err != nil {
		t.Fatal(err)
	}
	for idx, match := range matches {
		t.Run(fmt.Sprintf("%03d-%s", idx+1, match), func(t *testing.T) {
			testData, err := os.ReadFile(match)
			if err != nil {
				t.Fatalf("could not read file '%v': %v", match, err)
			}
			tests := strings.Split(string(testData), "\n")

			checkFilename := fmt.Sprintf("%s.out", match)
			checkData, err := os.ReadFile(checkFilename)
			if err != nil {
				t.Fatalf("Error on ReadFile('%s'): %s", checkFilename, err.Error())
			}
			checks := strings.Split(string(checkData), "\n")

			if len(checks) != len(tests) {
				t.Fatal("Template lines != Checks lines")
			}

			for idx, test := range tests {
				if strings.TrimSpace(test) == "" {
					continue
				}
				if strings.TrimSpace(checks[idx]) == "" {
					t.Fatalf("[%s Line %d] Check is empty (must contain an regular expression).",
						match, idx+1)
				}

				_, err = pongo2.FromString(test)
				if err != nil {
					t.Fatalf("Error on FromString('%s'): %s", test, err.Error())
				}

				tpl, err := pongo2.FromBytes([]byte(test))
				if err != nil {
					t.Fatalf("Error on FromBytes('%s'): %s", test, err.Error())
				}

				_, err = tpl.ExecuteBytes(tplContext)
				if err == nil {
					t.Fatalf("[%s Line %d] Expected error for (got none): %s",
						match, idx+1, tests[idx])
				}

				re := regexp.MustCompile(fmt.Sprintf("^%s$", checks[idx]))
				if !re.MatchString(err.Error()) {
					t.Fatalf("[%s Line %d] Error for '%s' (err = '%s') does not match the (regexp-)check: %s",
						match, idx+1, test, err.Error(), checks[idx])
				}
			}
		})
	}
}

func TestCompilationErrors(t *testing.T) {
	// debug = true

	matches, err := filepath.Glob("./template_tests/*-compilation.err")
	if err != nil {
		t.Fatal(err)
	}
	for idx, match := range matches {
		t.Run(fmt.Sprintf("%03d-%s", idx+1, match), func(t *testing.T) {
			testData, err := os.ReadFile(match)
			if err != nil {
				t.Fatalf("could not read file '%v': %v", match, err)
			}
			tests := strings.Split(string(testData), "\n")

			checkFilename := fmt.Sprintf("%s.out", match)
			checkData, err := os.ReadFile(checkFilename)
			if err != nil {
				t.Fatalf("error on ReadFile('%s'): %s", checkFilename, err.Error())
			}
			checks := strings.Split(string(checkData), "\n")

			if len(checks) != len(tests) {
				t.Fatal("Template lines != Checks lines")
			}

			for idx, test := range tests {
				if strings.TrimSpace(test) == "" {
					continue
				}
				if strings.TrimSpace(checks[idx]) == "" {
					t.Fatalf("[%s Line %d] Check is empty (must contain an regular expression).",
						match, idx+1)
				}

				_, err = pongo2.FromString(test)
				if err == nil {
					t.Fatalf("[%s | Line %d] Expected error for (got none): %s", match, idx+1, tests[idx])
				}
				re := regexp.MustCompile(fmt.Sprintf("^%s$", checks[idx]))
				if !re.MatchString(err.Error()) {
					t.Fatalf("[%s | Line %d] Error for '%s' (err = '%s') does not match the (regexp-)check: %s",
						match, idx+1, test, err.Error(), checks[idx])
				}
			}
		})
	}
}

func TestBaseDirectory(t *testing.T) {
	mustStr := "Hello from template_tests/base_dir_test/"

	fs := pongo2.MustNewLocalFileSystemLoader("")
	s := pongo2.NewSet("test set with base directory", fs)
	s.Globals["base_directory"] = "template_tests/base_dir_test/"
	if err := fs.SetBaseDir(s.Globals["base_directory"].(string)); err != nil {
		t.Fatal(err)
	}

	matches, err := filepath.Glob("./template_tests/base_dir_test/subdir/*")
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range matches {
		match = strings.Replace(match, fmt.Sprintf("template_tests%cbase_dir_test%c", filepath.Separator, filepath.Separator), "", -1)

		tpl, err := s.FromFile(match)
		if err != nil {
			t.Fatal(err)
		}
		out, err := tpl.Execute(nil)
		if err != nil {
			t.Fatal(err)
		}
		if out != mustStr {
			t.Errorf("%s: out ('%s') != mustStr ('%s')", match, out, mustStr)
		}
	}
}

func BenchmarkCache(b *testing.B) {
	cacheSet := pongo2.NewSet("cache set", pongo2.MustNewLocalFileSystemLoader(""))
	for i := 0; i < b.N; i++ {
		tpl, err := cacheSet.FromCache("template_tests/complex.tpl")
		if err != nil {
			b.Fatal(err)
		}
		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheDebugOn(b *testing.B) {
	cacheDebugSet := pongo2.NewSet("cache set", pongo2.MustNewLocalFileSystemLoader(""))
	cacheDebugSet.Debug = true
	for i := 0; i < b.N; i++ {
		tpl, err := cacheDebugSet.FromFile("template_tests/complex.tpl")
		if err != nil {
			b.Fatal(err)
		}
		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteComplexWithSandboxActive(b *testing.B) {
	tpl, err := pongo2.FromFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileAndExecuteComplexWithSandboxActive(b *testing.B) {
	buf, err := os.ReadFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	preloadedTpl := string(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tpl, err := pongo2.FromString(preloadedTpl)
		if err != nil {
			b.Fatal(err)
		}

		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelExecuteComplexWithSandboxActive(b *testing.B) {
	tpl, err := pongo2.FromFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkExecuteComplexWithoutSandbox(b *testing.B) {
	s := pongo2.NewSet("set without sandbox", pongo2.MustNewLocalFileSystemLoader(""))
	tpl, err := s.FromFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileAndExecuteComplexWithoutSandbox(b *testing.B) {
	buf, err := os.ReadFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	preloadedTpl := string(buf)

	s := pongo2.NewSet("set without sandbox", pongo2.MustNewLocalFileSystemLoader(""))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tpl, err := s.FromString(preloadedTpl)
		if err != nil {
			b.Fatal(err)
		}

		err = tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelExecuteComplexWithoutSandbox(b *testing.B) {
	s := pongo2.NewSet("set without sandbox", pongo2.MustNewLocalFileSystemLoader(""))
	tpl, err := s.FromFile("template_tests/complex.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := tpl.ExecuteWriterUnbuffered(tplContext, io.Discard)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkExecuteBlocksWithSandboxActive(b *testing.B) {
	blockNames := []string{"content", "more_content"}
	tpl, err := pongo2.FromFile("template_tests/block_render/block.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = tpl.ExecuteBlocks(tplContext, blockNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteBlocksDeepWithSandboxActive(b *testing.B) {
	blockNames := []string{"body", "more_content"}
	tpl, err := pongo2.FromFile("template_tests/block_render/deep.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = tpl.ExecuteBlocks(tplContext, blockNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteBlocksWithEmptyBlocksSandboxActive(b *testing.B) {
	blockNames := []string{}
	tpl, err := pongo2.FromFile("template_tests/block_render/block.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = tpl.ExecuteBlocks(tplContext, blockNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteBlocksWithoutSandbox(b *testing.B) {
	blockNames := []string{"content", "more_content"}
	s := pongo2.NewSet("set without sandbox", pongo2.MustNewLocalFileSystemLoader(""))
	tpl, err := s.FromFile("template_tests/block_render/block.tpl")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = tpl.ExecuteBlocks(tplContext, blockNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

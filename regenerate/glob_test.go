// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regenerate_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/goresed/goresed/internal/testline"
	"github.com/goresed/goresed/regenerate"
	"golang.org/x/tools/imports"
)

func TestGlob(t *testing.T) {
	tests := []struct {
		line         string
		name         string
		inputFiles   map[string]string
		inputPattern string
		inputOpts    []regenerate.Option
		wantFiles    map[string]string
	}{
		{
			name: "Converting db files within foo and bar packages.",
			line: testline.New(),
			inputFiles: map[string]string{
				"a/b/c/foo/db.go": `
// Code generated by sqlc. DO NOT EDIT.

package foo

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}
`,
				"x/y/z/bar/db.go": `
// Code generated by sqlc. DO NOT EDIT.

package bar

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}
`,
			},
			inputPattern: "*/*/*/*/db.go",
			inputOpts: []regenerate.Option{
				regenerate.ReplaceString(
					"// Code generated by sqlc. DO NOT EDIT.",
					"// Code generated by sqlc. DO NOT EDIT.\n// Code regenerated by goresed. DO NOT EDIT.",
				),
				regenerate.ReplaceRegexp(
					regexp.MustCompile(`(?ms)^\nimport \(.*`),
					`
import (
  "context"

  "github.com/your/customsql"
)

func New(dbtxf func(context.Context) customsql.DBTx) *Queries {
  return &Queries{database: dbtxf}
}

type Queries struct {
  database func(context.Context) customsql.DBTx
}
`,
				),
			},
			wantFiles: map[string]string{
				"a/b/c/foo/db.go": `
// Code generated by sqlc. DO NOT EDIT.
// Code regenerated by goresed. DO NOT EDIT.

package foo

import (
	"context"

	"github.com/your/customsql"
)

func New(dbtxf func(context.Context) customsql.DBTx) *Queries {
	return &Queries{database: dbtxf}
}

type Queries struct {
	database func(context.Context) customsql.DBTx
}
`,
				"x/y/z/bar/db.go": `
// Code generated by sqlc. DO NOT EDIT.
// Code regenerated by goresed. DO NOT EDIT.

package bar

import (
	"context"

	"github.com/your/customsql"
)

func New(dbtxf func(context.Context) customsql.DBTx) *Queries {
	return &Queries{database: dbtxf}
}

type Queries struct {
	database func(context.Context) customsql.DBTx
}
`,
			},
		},
		{
			name: "Converting query files within foo and bar packages.",
			line: testline.New(),
			inputFiles: map[string]string{
				"a/b/c/foo/query.sql.go": `
// Code generated by sqlc. DO NOT EDIT.
// source: query.sql

package foo

import "context"

const countFoos = "SELECT count(*) FROM foos;"

func (q *Queries) CountFoos(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx)
	var count int64
	err := row.Scan(&count)
	return count, err
}
`,
				"x/y/z/bar/query.sql.go": `
// Code generated by sqlc. DO NOT EDIT.
// source: query.sql

package bar

import "context"

const countBars = "SELECT count(*) FROM bars;"

func (q *Queries) CountBars(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx)
	var count int64
	err := row.Scan(&count)
	return count, err
}
`,
			},
			inputPattern: "*/*/*/*/query.sql.go",
			inputOpts: []regenerate.Option{
				regenerate.ReplaceString(
					"// Code generated by sqlc. DO NOT EDIT.\n// source: query.sql",
					"// Code generated by sqlc. DO NOT EDIT.\n// Code regenerated by goresed. DO NOT EDIT.\n// source: query.sql",
				),
				regenerate.ReplaceString(
					".db.",
					".database(ctx).",
				),
				regenerate.ReplaceString(
					"QueryContext",
					"Query",
				),
				regenerate.ReplaceString(
					"",
					"",
				),
				regenerate.ReplaceString(
					"QueryRowContext",
					"QueryRow",
				),
			},
			wantFiles: map[string]string{
				"a/b/c/foo/query.sql.go": `
// Code generated by sqlc. DO NOT EDIT.
// Code regenerated by goresed. DO NOT EDIT.
// source: query.sql

package foo

import "context"

const countFoos = "SELECT count(*) FROM foos;"

func (q *Queries) CountFoos(ctx context.Context) (int64, error) {
	row := q.database(ctx).QueryRow(ctx)
	var count int64
	err := row.Scan(&count)
	return count, err
}
`,
				"x/y/z/bar/query.sql.go": `
// Code generated by sqlc. DO NOT EDIT.
// Code regenerated by goresed. DO NOT EDIT.
// source: query.sql

package bar

import "context"

const countBars = "SELECT count(*) FROM bars;"

func (q *Queries) CountBars(ctx context.Context) (int64, error) {
	row := q.database(ctx).QueryRow(ctx)
	var count int64
	err := row.Scan(&count)
	return count, err
}
`,
			},
		},
	}

	direcotry, err := ioutil.TempDir("", "test_goresed")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(direcotry)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name+"/"+tt.line, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(direcotry, strings.Split(tt.line, ":")[1])

			for pth, file := range tt.inputFiles {
				err := os.MkdirAll(filepath.Dir(filepath.Join(dir, pth)), os.ModePerm)
				if err != nil {
					t.Fatalf("\nos make all dirs/%s\nerror: %s", tt.line, err)
				}

				f, err := os.OpenFile(filepath.Join(dir, pth), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
				if err != nil {
					t.Fatalf("\nos file create/%s\nerror: %s", tt.line, err)
				}

				_, err = f.Write([]byte(file))
				if err != nil {
					t.Fatalf("\nos file %s write/%s\nerror: %s", f.Name(), tt.line, err)
				}

				f.Close()
				if err != nil {
					t.Fatalf("\nos file %s close/%s\nerror: %s", f.Name(), tt.line, err)
				}
			}

			pattern := filepath.Join(dir, tt.inputPattern)

			var opts []regenerate.Option

			opts = append(opts, tt.inputOpts...)

			gofmt := imports.Options{
				Fragment:  true,
				Comments:  true,
				TabIndent: true,
				TabWidth:  8,
			}

			opts = append(opts, regenerate.WithGofmt(&gofmt))

			err := regenerate.Glob(pattern, opts...)
			if err != nil {
				t.Fatalf("\nregenerate/%s\nerror: %s", tt.line, err)
			}

			var files []*os.File
			var paths []string

			err = filepath.Walk(dir, func(pth string, info os.FileInfo, _ error) error {
				if info.IsDir() {
					return nil
				}
				f, err := os.Open(pth)
				if err != nil {
					return err
				}
				files = append(files, f)
				paths = append(paths, strings.TrimPrefix(pth, dir))
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}

			for _, f := range files {
				defer f.Close()
			}

			if len(files) != len(tt.wantFiles) {
				t.Errorf("\nfiles/%s\nwant files: %d\nget files/%d\n%+v", tt.line, len(files), len(tt.wantFiles), files)
			}

		filesLoop:
			for _, f := range files {
				get, err := ioutil.ReadAll(f)
				if err != nil {
					t.Fatal(err)
				}

				getPath := strings.TrimPrefix(f.Name(), dir+"/")

				for wantPath, wantFile := range tt.wantFiles {
					if wantPath != getPath {
						continue
					}

					want := strings.TrimSpace(wantFile)
					get := strings.TrimSpace(string(get))
					name := filepath.Base(getPath)

					if want != get && testing.Verbose() {
						t.Errorf(
							"\ntest: file %[2]s diff/%[1]s:\n%[3]s\nfile %[2]s want/%[1]s:\n%[4]s\nfile %[2]s get/%[1]s:\n%[5]s",
							tt.line, name, cmp.Diff(want, get), want, get,
						)
					} else if want != get {
						t.Errorf("\ntest: file %s diff/%s:\n%s", name, tt.line, cmp.Diff(want, get))
					}

					continue filesLoop
				}

				t.Errorf("unexpected file %q", getPath)
			}
		})
	}
}

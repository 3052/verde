package main

import (
   "bytes"
   "flag"
   "fmt"
   "go/ast"
   "go/doc"
   "go/format"
   "go/parser"
   "go/token"
   "os"
   "sort"
)

func main() {
   writeFlag := flag.Bool("w", false, "Write result to the source file instead of stdout")
   flag.Parse()

   if flag.NArg() < 1 {
      fmt.Println("Usage: godocorder [-w] <file.go>")
      os.Exit(1)
   }
   filename := flag.Arg(0)

   // 1. Read file and format it first.
   // Formatting first ensures exactly one declaration per line,
   // making our chunking math highly reliable.
   originalSrc, err := os.ReadFile(filename)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }
   src, err := format.Source(originalSrc)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Source has syntax errors: %v\n", err)
      os.Exit(1)
   }

   // 2. Parse the AST
   fset := token.NewFileSet()
   f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
      os.Exit(1)
   }

   // 3. Generate go/doc package to find official ordering.
   // We use doc.NewFromFiles to avoid the deprecated go/ast.Package.
   dpkg, err := doc.NewFromFiles(fset, []*ast.File{f}, "", doc.AllDecls|doc.PreserveAST)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error generating doc package: %v\n", err)
      os.Exit(1)
   }

   // 4. Calculate the new rank for each declaration
   type order struct{ cat, rank int }
   rankMap := make(map[int]order)
   rankCounter := 0

   astDeclIndex := make(map[ast.Decl]int)
   for i, d := range f.Decls {
      astDeclIndex[d] = i
   }

   add := func(cat int, d ast.Decl) {
      if idx, ok := astDeclIndex[d]; ok {
         if _, exists := rankMap[idx]; !exists {
            rankMap[idx] = order{cat, rankCounter}
            rankCounter++
         }
      }
   }

   // Cat 0: Imports always go first
   for i, d := range f.Decls {
      if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
         rankMap[i] = order{0, rankCounter}
         rankCounter++
      }
   }

   // Cat 2-5: Standard go doc order
   for _, v := range dpkg.Consts {
      add(2, v.Decl)
   }
   for _, v := range dpkg.Vars {
      add(3, v.Decl)
   }
   for _, fn := range dpkg.Funcs {
      add(4, fn.Decl)
   }
   for _, t := range dpkg.Types {
      add(5, t.Decl)
      // Group associated items with their type
      for _, v := range t.Consts {
         add(5, v.Decl)
      }
      for _, v := range t.Vars {
         add(5, v.Decl)
      }
      for _, fn := range t.Funcs {
         add(5, fn.Decl)
      }
      for _, m := range t.Methods {
         add(5, m.Decl)
      }
   }

   // Cat 1: Unranked items (init(), blank vars) right below imports
   for i := range f.Decls {
      if _, exists := rankMap[i]; !exists {
         rankMap[i] = order{1, rankCounter}
         rankCounter++
      }
   }

   // 5. Identify the "Header" end (Package decl + preceding comments)
   headerEnd := fset.Position(f.Name.End()).Offset
   headerEndLine := fset.Position(f.Name.End()).Line
   // Extend headerEnd past any trailing comments on the package line
   for _, cg := range f.Comments {
      for _, c := range cg.List {
         if fset.Position(c.Pos()).Line == headerEndLine {
            if e := fset.Position(c.End()).Offset; e > headerEnd {
               headerEnd = e
            }
         }
      }
   }
   // Advance past the newline
   headerEnd = advanceToNewline(src, headerEnd)

   // 6. Slice the code into physical chunks per declaration
   type chunk struct {
      code []byte
      cat  int
      rank int
      orig int
   }
   var chunks []chunk
   cursor := headerEnd

   for i, decl := range f.Decls {
      endOffset := fset.Position(decl.End()).Offset
      endLine := fset.Position(decl.End()).Line

      // Find trailing comments on the same line to keep them attached
      for _, cg := range f.Comments {
         for _, c := range cg.List {
            if fset.Position(c.Pos()).Line == endLine {
               if cEnd := fset.Position(c.End()).Offset; cEnd > endOffset {
                  endOffset = cEnd
               }
            }
         }
      }

      endOffset = advanceToNewline(src, endOffset)

      // Create a chunk of text for this declaration (includes preceding blank lines/comments)
      if endOffset > cursor {
         chunks = append(chunks, chunk{
            code: src[cursor:endOffset],
            cat:  rankMap[i].cat,
            rank: rankMap[i].rank,
            orig: i,
         })
         cursor = endOffset
      }
   }

   // The rest of the file (trailing EOF comments)
   footer := src[cursor:]

   // 7. Sort the text chunks based on their computed doc rank
   sort.SliceStable(chunks, func(i, j int) bool {
      if chunks[i].cat != chunks[j].cat {
         return chunks[i].cat < chunks[j].cat
      }
      return chunks[i].rank < chunks[j].rank
   })

   // 8. Reassemble the file
   var buf bytes.Buffer
   buf.Write(src[:headerEnd])
   for _, c := range chunks {
      buf.Write(c.code)
   }
   buf.Write(footer)

   // 9. Format one last time to ensure perfect spacing
   finalOut, err := format.Source(buf.Bytes())
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error formatting reassembled source: %v\n", err)
      os.Exit(1)
   }

   // 10. Write output
   if *writeFlag {
      if err := os.WriteFile(filename, finalOut, 0644); err != nil {
         fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
         os.Exit(1)
      }
      fmt.Printf("Reordered %s successfully.\n", filename)
   } else {
      os.Stdout.Write(finalOut)
   }
}

// advanceToNewline advances an offset to the character immediately following the next \n
func advanceToNewline(src []byte, offset int) int {
   for offset < len(src) && src[offset] != '\n' {
      offset++
   }
   if offset < len(src) && src[offset] == '\n' {
      offset++
   }
   return offset
}

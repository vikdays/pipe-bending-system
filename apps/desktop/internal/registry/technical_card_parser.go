package registry

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type cardNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Text     string     `xml:",chardata"`
	Children []cardNode `xml:",any"`
}
type cardTable struct {
	Index   int        `json:"index"`
	Caption string     `json:"caption"`
	Rows    [][]string `json:"rows"`
}
type TechnicalCardRow struct {
	Key        string             `json:"key"`
	Input      ConfigurationInput `json:"input"`
	SourceJSON string             `json:"sourceJson"`
	Warnings   []string           `json:"warnings"`
	ExistingID string             `json:"existingId"`
}
type TechnicalCardPreview struct {
	ID           string             `json:"id"`
	DocumentID   string             `json:"documentId"`
	OriginalName string             `json:"originalName"`
	SHA256       string             `json:"sha256"`
	Machine      Machine            `json:"machine"`
	Rows         []TechnicalCardRow `json:"rows"`
	Warnings     []string           `json:"warnings"`
}

func nodeText(n cardNode) string {
	if n.XMLName.Local == "t" {
		return n.Text
	}
	var parts []string
	for _, c := range n.Children {
		v := nodeText(c)
		if v != "" {
			parts = append(parts, v)
		}
	}
	separator := ""
	if n.XMLName.Local == "tc" {
		separator = " "
	}
	return strings.Join(parts, separator)
}
func cleanText(s string) string { return strings.Join(strings.Fields(s), " ") }
func nodeChild(n cardNode, name string) *cardNode {
	for i := range n.Children {
		if n.Children[i].XMLName.Local == name {
			return &n.Children[i]
		}
	}
	return nil
}
func docxTables(file *os.File) ([]cardTable, string, error) {
	info, e := file.Stat()
	if e != nil {
		return nil, "", invalid("Не удалось прочитать DOCX.")
	}
	z, e := zip.NewReader(file, info.Size())
	if e != nil {
		return nil, "", invalid("Нужен документ .docx без пароля.")
	}
	if len(z.File) > 2048 {
		return nil, "", invalid("Слишком много файлов внутри DOCX.")
	}
	var total uint64
	var data []byte
	headerText := ""
	for _, f := range z.File {
		if f.UncompressedSize64 > 128<<20 || total > (128<<20)-f.UncompressedSize64 {
			return nil, "", invalid("Распакованный DOCX превышает 128 МБ.")
		}
		total += f.UncompressedSize64
		if strings.Contains(strings.ToLower(f.Name), "vbaproject") {
			return nil, "", invalid("Документы с макросами не поддерживаются.")
		}
		if strings.HasPrefix(f.Name, "word/header") && strings.HasSuffix(f.Name, ".xml") {
			if f.UncompressedSize64 > 1<<20 {
				return nil, "", invalid("Колонтитул DOCX слишком велик.")
			}
			r, err := f.Open()
			if err != nil {
				return nil, "", invalid("DOCX повреждён.")
			}
			d := xml.NewDecoder(io.LimitReader(r, (1<<20)+1))
			for {
				token, err := d.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					r.Close()
					return nil, "", invalid("Повреждён колонтитул DOCX.")
				}
				if s, ok := token.(xml.CharData); ok {
					headerText += string(s)
				}
				if len(headerText) > 128<<10 {
					r.Close()
					return nil, "", invalid("Слишком много текста в колонтитулах DOCX.")
				}
			}
			r.Close()
		}
		if f.Name == "word/document.xml" {
			if f.UncompressedSize64 > 12<<20 {
				return nil, "", invalid("Текст DOCX превышает 12 МБ.")
			}
			r, err := f.Open()
			if err != nil {
				return nil, "", invalid("DOCX повреждён.")
			}
			data, e = io.ReadAll(io.LimitReader(r, (12<<20)+1))
			r.Close()
			if e != nil || len(data) > 12<<20 {
				return nil, "", invalid("Не удалось прочитать текст DOCX.")
			}
		}
	}
	if len(data) == 0 {
		return nil, "", invalid("В DOCX нет основного текста.")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	depth, count := 0, 0
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, "", invalid("Повреждён XML документа.")
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
			count++
			if depth > 128 || count > 200000 {
				return nil, "", invalid("Структура DOCX слишком сложна.")
			}
		case xml.EndElement:
			depth--
		}
	}
	var root cardNode
	if xml.Unmarshal(data, &root) != nil {
		return nil, "", invalid("DOCX не удалось разобрать.")
	}
	body := nodeChild(root, "body")
	if body == nil {
		return nil, "", invalid("В DOCX отсутствует текст документа.")
	}
	tables := []cardTable{}
	paragraphs := []string{}
	caption := ""
	for _, n := range body.Children {
		switch n.XMLName.Local {
		case "p":
			s := cleanText(nodeText(n))
			if s != "" {
				paragraphs = append(paragraphs, s)
				caption = s
			}
		case "tbl":
			t := cardTable{Index: len(tables) + 1, Caption: caption, Rows: [][]string{}}
			for _, row := range n.Children {
				if row.XMLName.Local != "tr" {
					continue
				}
				cells := []string{}
				for _, cell := range row.Children {
					if cell.XMLName.Local != "tc" {
						continue
					}
					span := 1
					if pr := nodeChild(cell, "tcPr"); pr != nil {
						if grid := nodeChild(*pr, "gridSpan"); grid != nil {
							for _, a := range grid.Attrs {
								if a.Name.Local == "val" {
									span, _ = strconv.Atoi(a.Value)
								}
							}
						}
					}
					if span < 1 || span > 128 {
						return nil, "", invalid("Некорректное объединение ячеек DOCX.")
					}
					s := cleanText(nodeText(cell))
					for i := 0; i < span; i++ {
						cells = append(cells, s)
					}
				}
				if len(cells) > 128 || len(t.Rows) >= 5000 {
					return nil, "", invalid("Таблица DOCX слишком велика.")
				}
				t.Rows = append(t.Rows, cells)
			}
			tables = append(tables, t)
		}
	}
	return tables, strings.Join(paragraphs, "\n") + "\n" + headerText, nil
}

var diameterCaption = regexp.MustCompile(`(?i)(?:из труб|труб)\s*(?:Ø|ø|диаметром)?\s*(219|273|325|426|530|720|1020)\s*мм`)

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}
func numericCell(value string) *float64 {
	v := strings.TrimSpace(strings.TrimRight(value, "* "))
	n, e := parseNumber(v, "")
	if e != nil {
		return nil
	}
	return n
}
func ParseTechnicalCard(file *os.File, machine Machine) ([]TechnicalCardRow, []string, error) {
	tables, text, e := docxTables(file)
	if e != nil {
		return nil, nil, e
	}
	model := heading(machine.Model)
	scope := heading(text)
	if !strings.Contains(scope, model) {
		return nil, nil, invalid("Текст карты не подтверждает выбранную модель станка.")
	}
	output := []TechnicalCardRow{}
	for _, t := range tables {
		match := diameterCaption.FindStringSubmatch(t.Caption)
		if match == nil || !strings.Contains(strings.ToLower(t.Caption), "режим") {
			continue
		}
		diameter, _ := strconv.ParseFloat(match[1], 64)
		if model == "гт532" {
			if len(t.Rows) == 0 || len(t.Rows[0]) != 8 || !strings.Contains(heading(cell(t.Rows[0], 0)), "уголгибки") {
				continue
			}
			angle, strength := "", ""
			for i, row := range t.Rows[1:] {
				if len(row) != 8 {
					continue
				}
				if cell(row, 0) != "" {
					angle = cell(row, 0)
				}
				if cell(row, 1) != "" {
					strength = cell(row, 1)
				}
				if numericCell(angle) == nil || numericCell(cell(row, 2)) == nil {
					continue
				}
				p := []ConfigurationParameter{{Name: "Класс прочности трубы", Value: strength}, {Name: "Единичный угол гибки", Value: cell(row, 2), Unit: "град"}, {Name: "Количество гибов", Value: cell(row, 3), Unit: "шт."}, {Name: "Передвижка", Value: cell(row, 4), Unit: "мм"}, {Name: "Количество передвижек", Value: cell(row, 5), Unit: "шт."}, {Name: "Рабочая зона L2", Value: cell(row, 6), Unit: "мм"}, {Name: "Минимальная длина трубы", Value: cell(row, 7), Unit: "мм"}}
				// L1/L3 are read from the dedicated table, never inferred from equipment ranges.
				for _, straight := range tables {
					if !strings.Contains(strings.ToLower(straight.Caption), "длины прямых") {
						continue
					}
					if len(straight.Rows) >= 2 {
						p = append(p, ConfigurationParameter{Name: cell(straight.Rows[0], 0), Value: cell(straight.Rows[1], 0), Unit: "мм"}, ConfigurationParameter{Name: cell(straight.Rows[0], 1), Value: cell(straight.Rows[1], 1), Unit: "мм"})
					}
				}
				output = append(output, makeCardRow(machine, diameter, "1ГО", angle, t, i+2, 0, p, numericCell(cell(row, 7))))
			}
		} else if model == "гт1022" {
			var angles, types []string
			angleRow := 0
			for i, row := range t.Rows {
				if strings.Contains(heading(cell(row, 0)), "суммарныйугол") {
					angles = row
					angleRow = i
				}
				if strings.Contains(heading(cell(row, 0)), "типотвода") {
					types = row
				}
			}
			for col := 1; col < len(angles); col++ {
				if numericCell(angles[col]) == nil {
					continue
				}
				kind := ""
				switch heading(cell(types, col)) {
				case "тип1":
					kind = "1ГО"
				case "тип2":
					kind = "2ГО"
				}
				if kind == "" {
					return nil, nil, invalid("Не удалось определить тип отвода в объединённых ячейках ТК.")
				}
				params := []ConfigurationParameter{}
				var minimum *float64
				for _, row := range t.Rows[angleRow+1:] {
					label, value := cell(row, 0), cell(row, col)
					if label == "" || value == "" || strings.HasPrefix(label, "*") {
						continue
					}
					unit := ""
					h := heading(label)
					if strings.Contains(h, "мм") {
						unit = "мм"
					}
					if strings.Contains(h, "град") {
						unit = "град"
					}
					params = append(params, ConfigurationParameter{Name: label, Value: value, Unit: unit})
					if strings.HasPrefix(h, "минимальнаядлинатрубы") {
						minimum = numericCell(value)
					}
				}
				output = append(output, makeCardRow(machine, diameter, kind, angles[col], t, angleRow+1, col+1, params, minimum))
			}
		}
	}
	if len(output) == 0 {
		return nil, nil, invalid("Таблицы двух изученных форм ТК не распознаны. Оригинал не преобразуется в произвольные режимы; используйте ручное создание конфигурации.")
	}
	if len(output) > 1000 {
		return nil, nil, invalid("Слишком много конфигураций в карте.")
	}
	// A cited GOST's year is not the card's year. Prefer the explicit card code
	// from its header, otherwise a title-page place/year line.
	code := regexp.MustCompile(`ТК\s*([0-9]{4}-[0-9]{3}-(20[0-9]{2}))`).FindStringSubmatch(text)
	version := "Редакция и дата не указаны"
	if code != nil {
		version = "ТК " + code[1] + " — редакция не указана"
	} else if titleYear := regexp.MustCompile(`г\.\s*Томск\s*(20[0-9]{2})`).FindStringSubmatch(text); titleYear != nil {
		version = titleYear[1] + " — редакция не указана"
	}
	for i := range output {
		output[i].Input.Version = version
		for _, reference := range tables {
			if !strings.Contains(heading(reference.Caption), "справочныепараметрытруб") {
				continue
			}
			for rowIndex, values := range reference.Rows {
				d := numericCell(cell(values, 0))
				if d == nil || *d != output[i].Input.DiameterMm || len(values) != 6 {
					continue
				}
				for _, field := range []struct {
					index      int
					name, unit string
				}{{1, "Толщина стенки — справочная таблица", "мм"}, {2, "Класс прочности — справочная таблица", ""}, {3, "Условный предел текучести — справочная таблица", "Н/мм²"}, {4, "Минимальный радиус гибки — справочная таблица", "м"}, {5, "Целевые углы — справочная таблица", "град"}} {
					output[i].Input.Parameters = append(output[i].Input.Parameters, ConfigurationParameter{Name: field.name, Value: values[field.index], Unit: field.unit})
				}
				var source map[string]any
				_ = json.Unmarshal([]byte(output[i].SourceJSON), &source)
				source["referenceTable"] = map[string]any{"table": reference.Index, "caption": reference.Caption, "row": rowIndex + 1, "cells": values}
				output[i].SourceJSON = jsonText(source)
			}
		}
	}
	return output, []string{"Значения и сноски перенесены из таблиц без пересчёта. Каждая конфигурация требует проверки по оригиналу перед подготовкой задания.", "Параметры H и стрелы прогиба не смешиваются и автоматически не вычисляются.", "Год документа не подтверждает его утверждение или текущую редакцию."}, nil
}
func makeCardRow(m Machine, diameter float64, kind, angle string, t cardTable, row, col int, p []ConfigurationParameter, minimum *float64) TechnicalCardRow {
	footnotes := []string{}
	for _, r := range t.Rows {
		if len(r) == 0 {
			continue
		}
		s := cell(r, 0)
		if strings.HasPrefix(s, "*") {
			footnotes = append(footnotes, s)
		}
	}
	raw := []any{}
	if col == 0 {
		raw = append(raw, t.Rows[0], t.Rows[row-1])
	} else {
		for i, r := range t.Rows {
			raw = append(raw, map[string]any{"row": i + 1, "label": cell(r, 0), "value": cell(r, col-1)})
		}
	}
	source := map[string]any{"parserVersion": 1, "table": t.Index, "caption": t.Caption, "row": row, "column": col, "cells": raw, "footnotes": footnotes}
	for i := range p {
		if len([]rune(p[i].Name)) > 160 {
			p[i].Name = string([]rune(p[i].Name)[:157]) + "…"
		}
	}
	name := fmt.Sprintf("%s · Ø%g · %s · %s°", m.Name, diameter, kind, angle)
	for _, parameter := range p {
		h := heading(parameter.Name)
		if strings.Contains(h, "единичныйугол") {
			name += " · гиб " + parameter.Value + "°"
		}
		if strings.HasPrefix(h, "передвижка") {
			name += " · шаг " + parameter.Value + " мм"
		}
	}
	input := ConfigurationInput{Name: name, Version: "Редакция не указана", Machine: m.Name, MachineID: m.ID, TechnicalCardReference: t.Caption, DiameterMm: diameter, BendType: kind, TargetAngle: strings.ReplaceAll(angle, ",", "."), MinPipeLengthMm: minimum, Parameters: p, Notes: strings.Join(footnotes, "\n")}
	warnings := []string{}
	if minimum == nil {
		warnings = append(warnings, "Минимальная длина не распознана числом.")
	}
	for _, v := range p {
		if strings.Contains(v.Value, "*") {
			warnings = append(warnings, "В значениях есть сноски: проверьте условия применения распорок и оснастки.")
			break
		}
	}
	// This is a consistency diagnostic, not a replacement for the source's values.
	var l1, l2, l3 *float64
	for _, parameter := range p {
		h := heading(parameter.Name)
		if strings.Contains(h, "l1") {
			l1 = numericCell(parameter.Value)
		}
		if strings.Contains(h, "l2") {
			l2 = numericCell(parameter.Value)
		}
		if strings.Contains(h, "l3") && !strings.Contains(h, "шва") {
			l3 = numericCell(parameter.Value)
		}
	}
	if minimum != nil && l1 != nil && l2 != nil && l3 != nil && *minimum != *l1+*l2+*l3 {
		warnings = append(warnings, "Проверьте источник: указанная минимальная длина отличается от суммы L1, L2 и L3. Значения не исправлены автоматически.")
		input.Notes += "\n" + warnings[len(warnings)-1]
	}
	return TechnicalCardRow{Key: fmt.Sprintf("t%d-r%d-c%d", t.Index, row, col), Input: input, SourceJSON: jsonText(source), Warnings: warnings}
}
func cardSource(fileName, sha, raw string) string {
	var s map[string]any
	_ = json.Unmarshal([]byte(raw), &s)
	s["fileName"] = fileName
	s["sha256"] = sha
	return jsonText(s)
}

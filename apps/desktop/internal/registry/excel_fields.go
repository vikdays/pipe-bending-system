package registry

import (
	"strings"
	"unicode"
)

type fieldSpec struct {
	ImportField
	aliases []string
}

var importFields = []fieldSpec{
	{ImportField{"inventoryNumber", "Инвентарный номер записи", false, []string{}}, []string{"Инвентарный номер", "Инвентарный номер записи", "inventoryNumber"}},
	{ImportField{"factoryNumber", "Заводской номер трубы", true, []string{}}, []string{"Номер трубы", "Заводской номер трубы", "Заводской номер", "Зав. номер трубы", "factoryNumber"}},
	{ImportField{"certificateReference", "Реквизиты сертификата", false, []string{}}, []string{"№ Сертификата, дата сертификата, количество листов", "Реквизиты сертификата", "Сертификат", "Номер сертификата"}},
	{ImportField{"manufacturer", "Завод-изготовитель", false, []string{}}, []string{"Завод изготовитель", "Завод-изготовитель трубы", "Изготовитель", "manufacturer"}},
	{ImportField{"incomingInspectionReference", "Документ входного контроля", false, []string{}}, []string{"Реквизиты документа входного контроля", "Входной контроль"}},
	{ImportField{"diameterMm", "Диаметр", false, []string{"mm", "m"}}, []string{"Диаметр трубы, мм", "Диаметр, мм", "Наружный диаметр, мм", "diameterMm"}},
	{ImportField{"wallThicknessMm", "Толщина стенки", false, []string{"mm", "m"}}, []string{"Толщина стенки, мм", "Стенка, мм", "wallThicknessMm"}},
	{ImportField{"lengthMm", "Длина трубы", false, []string{"mm", "m"}}, []string{"Длина трубы, мм", "Длина, мм", "Длина трубы, м", "lengthMm"}},
	{ImportField{"massKg", "Масса трубы", false, []string{"kg", "t"}}, []string{"Масса трубы", "Масса трубы, кг", "Масса, кг", "Масса, т", "massKg"}},
	{ImportField{"strengthClass", "Класс прочности", false, []string{}}, []string{"Класс прочности", "strengthClass"}},
	{ImportField{"steelGrade", "Марка стали", false, []string{}}, []string{"Марка стали", "steelGrade"}},
	{ImportField{"heatNumber", "Номер плавки", false, []string{}}, []string{"Номер плавки", "№ плавки", "heatNumber"}},
	{ImportField{"carbonEquivalentPct", "Эквивалент углерода", false, []string{"percent"}}, []string{"Эквивалент углерода, %", "Эквивалент углерода", "Сeq", "Ceq", "carbonEquivalentPct"}},
	{ImportField{"pipeSpecification", "ТУ / стандарт трубы", false, []string{}}, []string{"ТУ трубы", "ТУ на трубу", "Стандарт трубы", "pipeSpecification"}},
	{ImportField{"coatingType", "Тип покрытия", false, []string{}}, []string{"Тип изоляции", "Тип покрытия", "coatingType"}},
	{ImportField{"coatingSpecification", "ТУ покрытия", false, []string{}}, []string{"ТУ изоляции", "ТУ покрытия", "coatingSpecification"}},
	{ImportField{"testPressureMpa", "Испытательное давление", false, []string{"MPa"}}, []string{"Испытательное давление, МПа", "Гидроиспытания, МПа", "testPressureMpa"}},
	{ImportField{"minOperatingTempC", "Мин. температура стенки при эксплуатации", false, []string{"C"}}, []string{"Минимальная температура стенки при эксплуатации", "minOperatingTempC"}},
	{ImportField{"allowedCategory54", "Категория при 5,4 МПа", false, []string{}}, []string{"Допустимая категория участка для газопровода с Рраб=5.4 МПа", "Категория 5,4 МПа", "allowedCategory54"}},
	{ImportField{"allowedCategory74", "Категория при 7,4 МПа", false, []string{}}, []string{"Допустимая категория участка для газопровода с Рраб=7.4 МПа", "Категория 7,4 МПа", "allowedCategory74"}},
}

func heading(value string) string {
	return strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		if r == 'ё' {
			r = 'е'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value)
}
func textFields(p *PipeInput) map[string]*string {
	return map[string]*string{"inventoryNumber": &p.InventoryNumber, "factoryNumber": &p.FactoryNumber, "manufacturer": &p.Manufacturer, "incomingInspectionReference": &p.IncomingInspectionReference, "strengthClass": &p.StrengthClass, "steelGrade": &p.SteelGrade, "heatNumber": &p.HeatNumber, "pipeSpecification": &p.PipeSpecification, "coatingType": &p.CoatingType, "coatingSpecification": &p.CoatingSpecification, "allowedCategory54": &p.AllowedCategory54, "allowedCategory74": &p.AllowedCategory74}
}
func numberFields(p *PipeInput) map[string]**float64 {
	return map[string]**float64{"diameterMm": &p.DiameterMm, "wallThicknessMm": &p.WallThicknessMm, "lengthMm": &p.LengthMm, "massKg": &p.MassKg, "carbonEquivalentPct": &p.CarbonEquivalentPct, "testPressureMpa": &p.TestPressureMpa, "minOperatingTempC": &p.MinOperatingTempC}
}

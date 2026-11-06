package registry

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"changeme/internal/apperror"
)

func invalid(message string) error { return apperror.New(apperror.ValidationError, message) }

func normalizePipe(input PipeInput) (PipeInput, error) {
	for _, field := range []*string{
		&input.InventoryNumber, &input.FactoryNumber, &input.Manufacturer,
		&input.StrengthClass, &input.SteelGrade, &input.HeatNumber,
		&input.PipeSpecification, &input.CoatingType, &input.CoatingSpecification,
		&input.IncomingInspectionReference, &input.AllowedCategory54, &input.AllowedCategory74,
	} {
		*field = strings.TrimSpace(*field)
		if !validText(*field, 1000) {
			return input, invalid("Текстовое поле слишком длинное или содержит недопустимые символы.")
		}
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if !validText(input.Notes, 4000) {
		return input, invalid("Примечание должно содержать не более 4000 символов.")
	}
	if input.InventoryNumber == "" || !validText(input.InventoryNumber, 80) || input.FactoryNumber == "" || !validText(input.FactoryNumber, 160) {
		return input, invalid("Укажите инвентарный номер записи (до 80 символов) и заводской номер трубы (до 160 символов).")
	}
	for _, value := range []*float64{input.DiameterMm, input.WallThicknessMm, input.LengthMm, input.MassKg, input.TestPressureMpa} {
		if value != nil && (!finite(*value) || *value <= 0) {
			return input, invalid("Размеры, масса и испытательное давление должны быть положительными числами. Неизвестные значения оставьте пустыми.")
		}
	}
	if input.DiameterMm != nil && input.WallThicknessMm != nil && 2**input.WallThicknessMm >= *input.DiameterMm {
		return input, invalid("Удвоенная толщина стенки должна быть меньше наружного диаметра.")
	}
	if input.CarbonEquivalentPct != nil && (!finite(*input.CarbonEquivalentPct) || *input.CarbonEquivalentPct < 0 || *input.CarbonEquivalentPct > 100) {
		return input, invalid("Эквивалент углерода в процентах должен быть числом от 0 до 100.")
	}
	if input.MinOperatingTempC != nil && !finite(*input.MinOperatingTempC) {
		return input, invalid("Температура должна быть конечным числом.")
	}
	input.AttributesJSON = strings.TrimSpace(input.AttributesJSON)
	if input.AttributesJSON == "" {
		input.AttributesJSON = "{}"
	}
	var attributes map[string]json.RawMessage
	if len(input.AttributesJSON) > 65536 || json.Unmarshal([]byte(input.AttributesJSON), &attributes) != nil || attributes == nil {
		return input, invalid("Дополнительные сведения должны быть JSON-объектом размером не более 64 КБ.")
	}
	if len(input.CertificateIDs) > 100 {
		return input, invalid("Слишком много связанных сертификатов.")
	}
	seen := map[string]bool{}
	for _, id := range input.CertificateIDs {
		if id == "" || !validText(id, 100) || seen[id] {
			return input, invalid("Проверьте список связанных сертификатов.")
		}
		seen[id] = true
	}
	if input.NewCertificate != nil {
		certificate := *input.NewCertificate
		for _, field := range []*string{&certificate.Reference, &certificate.Number, &certificate.IssuedOn, &certificate.Issuer} {
			*field = strings.TrimSpace(*field)
			if !validText(*field, 1000) {
				return input, invalid("Проверьте реквизиты сертификата.")
			}
		}
		if certificate.Reference == "" {
			return input, invalid("Укажите полные реквизиты нового сертификата.")
		}
		if certificate.IssuedOn != "" {
			if _, err := time.Parse("2006-01-02", certificate.IssuedOn); err != nil {
				return input, invalid("Дата сертификата должна иметь формат ГГГГ-ММ-ДД.")
			}
		}
		if certificate.PageCount != nil && *certificate.PageCount <= 0 {
			return input, invalid("Количество листов должно быть положительным целым числом.")
		}
		input.NewCertificate = &certificate
	}
	return input, nil
}

func normalizeTask(input TaskInput) (TaskInput, error) {
	for _, field := range []*string{&input.Number, &input.PipeID, &input.BendNumber, &input.Machine, &input.MachineID, &input.ConfigurationID, &input.BendType} {
		*field = strings.TrimSpace(*field)
		if !validText(*field, 160) {
			return input, invalid("Поля задания должны содержать не более 160 символов.")
		}
	}
	if input.PipeID == "" || input.PipeRevision <= 0 {
		return input, invalid("Выберите конкретную трубу из актуального трубного журнала.")
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if !validText(input.Notes, 4000) {
		return input, invalid("Примечание задания должно содержать не более 4000 символов.")
	}
	if input.BendType != "" && input.BendType != "1ГО" && input.BendType != "2ГО" {
		return input, invalid("Выберите тип отвода: 1ГО или 2ГО.")
	}
	if input.ConfigurationID == "" {
		if input.ConfigurationRevision != 0 {
			return input, invalid("Проверьте выбранную конфигурацию.")
		}
	} else if input.ConfigurationRevision <= 0 {
		return input, invalid("Обновите список конфигураций.")
	}
	input.TargetAngle = strings.ReplaceAll(strings.TrimSpace(input.TargetAngle), ",", ".")
	if input.TargetAngle == "" {
		return input, nil
	}
	if len(input.TargetAngle) > 40 || strings.Trim(input.TargetAngle, "0123456789.") != "" {
		return input, invalid("Целевой угол должен быть положительным числом, например 9 или 9,5.")
	}
	angle, err := strconv.ParseFloat(input.TargetAngle, 64)
	if err != nil || !finite(angle) || angle <= 0 {
		return input, invalid("Целевой угол должен быть положительным конечным числом.")
	}
	input.TargetAngle = strconv.FormatFloat(angle, 'f', -1, 64)
	if input.PipeRevision <= 0 {
		return input, invalid("Обновите список труб и выберите запись снова.")
	}
	return input, nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func validText(value string, limit int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit && !strings.ContainsRune(value, 0)
}

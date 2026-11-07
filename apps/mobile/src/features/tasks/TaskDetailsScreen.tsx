import { useEffect, useState } from 'react';
import { ActivityIndicator, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useTaskRepository } from '@/storage/TaskStorageProvider';
import type { TransferredTask } from './taskTypes';
import { taskStyles as s } from './taskStyles';

function Field({ label, value }: { label: string; value: string | number | null }) {
  return <View style={s.field}><Text style={s.label}>{label}</Text><Text selectable style={s.text}>{value === null || value === '' ? 'Не указано' : String(value)}</Text></View>;
}
export function TaskDetailsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const repository = useTaskRepository();
  const [task, setTask] = useState<TransferredTask | null>(null);
  const [importedAt, setImportedAt] = useState('');
  const [nextId, setNextId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    setLoading(true); setTask(null); setError('');
    Promise.all([repository.get(typeof id === 'string' ? id : ''), repository.list()]).then(([result, list]) => {
      if (!active) return;
      if (!result) { setError('Задание не найдено в локальной базе.'); return; }
      setTask(result.task); setImportedAt(result.importedAt);
      const index = list.findIndex((t) => t.id === result.task.id.toLowerCase());
      setNextId(list[index + 1]?.id ?? null);
    }).catch(() => { if (active) setError('Не удалось прочитать карточку задания.'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [repository, id, attempt]);
  if (loading) return <View style={[s.screen, { justifyContent: 'center' }]}><ActivityIndicator size="large" /></View>;
  if (error || !task) return <View style={s.content}><Text accessibilityRole="alert" style={s.error}>{error}</Text>
    <Pressable accessibilityRole="button" style={s.button} onPress={() => setAttempt((n) => n + 1)}><Text style={s.buttonText}>Повторить</Text></Pressable></View>;
  const pipe = task.pipeSnapshot, config = task.configurationSnapshot, machine = task.machineSnapshot;
  return <SafeAreaView style={s.screen} edges={['bottom']}><ScrollView contentContainerStyle={s.content}>
    <Text style={s.title}>{task.number}</Text>
    <Text style={s.subtitle}>Очередь: {task.queuePosition} · Ревизия: {task.revision}{'\n'}Задание сохранено на этом устройстве.</Text>
    <View style={s.card}><Text style={s.cardTitle}>Задание</Text>
      <Field label="Планируемый номер отвода" value={task.bendNumber} /><Field label="Тип отвода" value={task.bendType} />
      <Field label="Целевой угол, град" value={task.targetAngleDeg} /><Field label="Примечание" value={task.notes} />
      <Field label="Подготовлено на ПК" value={task.preparedAt} /><Field label="Загружено на устройство" value={importedAt} />
    </View>
    <View style={s.card}><Text style={s.cardTitle}>Исходная труба</Text>
      {([
        ['Инвентарный номер', pipe.inventoryNumber], ['Заводской номер', pipe.factoryNumber], ['Изготовитель', pipe.manufacturer],
        ['Диаметр, мм', pipe.diameterMm], ['Толщина стенки, мм', pipe.wallThicknessMm], ['Длина, мм', pipe.lengthMm],
        ['Масса, кг', pipe.massKg], ['Класс прочности', pipe.strengthClass], ['Марка стали', pipe.steelGrade],
        ['Плавка', pipe.heatNumber], ['Углеродный эквивалент, %', pipe.carbonEquivalentPct], ['Нормативный документ трубы', pipe.pipeSpecification],
        ['Покрытие', pipe.coatingType], ['Документ покрытия', pipe.coatingSpecification], ['Давление испытания, МПа', pipe.testPressureMpa],
        ['Минимальная рабочая температура, °C', pipe.minOperatingTempC], ['Входной контроль', pipe.incomingInspectionReference],
        ['Категория 54', pipe.allowedCategory54], ['Категория 74', pipe.allowedCategory74], ['Примечание к трубе', pipe.notes],
      ] as [string, string | number | null][]).map(([label, value]) => <Field key={label} label={label} value={value} />)}
      <Field label="Сертификаты" value={pipe.certificates.map((c) => c.reference || c.number).join('\n')} />
      <Field label="Дополнительные сведения (JSON)" value={JSON.stringify(JSON.parse(pipe.attributesJson), null, 2)} />
    </View>
    <View style={s.card}><Text style={s.cardTitle}>Станок</Text><Field label="Наименование" value={machine.name} />
      <Field label="Модель" value={machine.model} /><Field label="Заводской номер" value={machine.serialNumber} />
      <Field label="Инвентарный номер" value={machine.inventoryNumber} /></View>
    <View style={s.card}><Text style={s.cardTitle}>Технологическая конфигурация</Text>
      <Field label="Название" value={config.name} /><Field label="Версия и ревизия" value={`${config.version} · ${config.revision}`} />
      <Field label="Технологическая карта" value={config.technicalCardReference} />
      <Field label="Минимальная длина трубы, мм" value={config.minPipeLengthMm} />
      {config.parameters.map((parameter, i) => <Field key={`${i}-${parameter.name}`} label={parameter.name} value={[parameter.value, parameter.unit].filter(Boolean).join(' ')} />)}
      <Field label="Примечание к конфигурации" value={config.notes} />
    </View>
    <Text style={s.subtitle}>Фактическая погода и температура будут вводиться при начале сеанса. Экран сеанса — следующий этап разработки.</Text>
    {nextId && <Pressable accessibilityRole="button" style={s.button} onPress={() => router.replace({ pathname: '/tasks/[id]', params: { id: nextId } })}><Text style={s.buttonText}>Следующее задание</Text></Pressable>}
  </ScrollView></SafeAreaView>;
}

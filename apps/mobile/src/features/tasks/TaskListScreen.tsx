import { ActivityIndicator, FlatList, Modal, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useTaskList } from './useTaskList';
import { taskStyles as s } from './taskStyles';

export function TaskListScreen() {
  const vm = useTaskList();
  return <SafeAreaView style={s.screen} edges={['bottom']}>
    <FlatList data={vm.tasks} keyExtractor={(task) => task.id}
      contentContainerStyle={s.content} refreshing={vm.loading} onRefresh={() => void vm.refresh()}
      ListHeaderComponent={<View style={{ gap: 16 }}>
        <Text style={s.title}>Задания машиниста</Text>
        <Text style={s.subtitle}>На устройстве: {vm.tasks.length} · Пакетов: {vm.packages}{'\n'}Данные доступны без Интернета.</Text>
        <Pressable accessibilityRole="button" disabled={vm.busy || vm.loading} style={[s.button, (vm.busy || vm.loading) && s.disabled]} onPress={() => void vm.choose()}>
          <Text style={s.buttonText}>{vm.busy ? 'Обработка…' : 'Импортировать задания'}</Text>
        </Pressable>
        {!!vm.error && <Text accessibilityRole="alert" style={s.error}>{vm.error}</Text>}
        {!!vm.message && <Text accessibilityRole="alert" style={s.message}>{vm.message}</Text>}
      </View>}
      ListEmptyComponent={vm.loading ? <ActivityIndicator size="large" /> : <View style={s.card}>
        <Text style={s.cardTitle}>Заданий пока нет</Text>
        <Text style={s.text}>Сохраните JSON-пакет с ПК в память телефона или на USB-накопитель. Нажмите «Импортировать задания» и выберите файл.</Text>
      </View>}
      renderItem={({ item }) => <Pressable accessibilityRole="button" accessibilityLabel={`Открыть задание ${item.number}`} style={s.card}
        onPress={() => router.push({ pathname: '/tasks/[id]', params: { id: item.id } })}>
        <Text style={s.label}>Очередь: {item.queuePosition} · Загружено</Text>
        <Text style={s.cardTitle}>{item.number} · отвод {item.bendNumber}</Text>
        <Text style={s.text}>Труба {item.factoryNumber}{'\n'}{item.machineModel} · целевой угол {item.targetAngleDeg}°</Text>
      </Pressable>} />
    <Modal visible={vm.preview !== null} animationType="slide" onRequestClose={vm.cancel}>
      <SafeAreaView style={s.screen}>
        <FlatList data={vm.preview?.data.tasks ?? []} keyExtractor={(task) => task.id} contentContainerStyle={s.content}
          ListHeaderComponent={<View style={{ gap: 12 }}><Text style={s.title}>Проверьте пакет</Text>
            <Text style={s.subtitle}>{vm.preview?.name}{'\n'}Заданий: {vm.preview?.data.tasks.length ?? 0}. Будет сохранён весь список в указанном порядке.</Text></View>}
          renderItem={({ item }) => <View style={s.card}><Text style={s.cardTitle}>{item.queuePosition}. {item.number}</Text>
            <Text style={s.text}>Отвод {item.bendNumber} · труба {item.pipeSnapshot.factoryNumber}{'\n'}{item.machineSnapshot.model} · {item.targetAngleDeg}°</Text></View>}
          ListFooterComponent={<View style={{ gap: 12 }}>
            <Pressable accessibilityRole="button" disabled={vm.busy} style={[s.button, vm.busy && s.disabled]} onPress={() => void vm.confirm()}>
              <Text style={s.buttonText}>{vm.busy ? 'Сохранение…' : `Сохранить задания (${vm.preview?.data.tasks.length ?? 0})`}</Text></Pressable>
            <Pressable accessibilityRole="button" disabled={vm.busy} style={[s.button, s.secondary]} onPress={vm.cancel}><Text style={[s.buttonText, s.secondaryText]}>Отмена</Text></Pressable>
          </View>} />
      </SafeAreaView>
    </Modal>
  </SafeAreaView>;
}

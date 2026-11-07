import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { TaskStorageProvider } from '@/storage/TaskStorageProvider';

export default function RootLayout() {
  return <TaskStorageProvider>
    <StatusBar style="dark" />
    <Stack screenOptions={{ headerStyle: { backgroundColor: '#ffffff' }, headerTintColor: '#182c43', contentStyle: { backgroundColor: '#f5f6f8' } }}>
      <Stack.Screen name="index" options={{ title: 'Контроль гибки' }} />
      <Stack.Screen name="tasks/[id]" options={{ title: 'Карточка задания' }} />
    </Stack>
  </TaskStorageProvider>;
}

import { useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';

export default function HomeScreen() {
  const [count, setCount] = useState(0);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>
        Цифровой контроль гибки
      </Text>

      <Text style={styles.description}>
        Первый экран приложения машиниста
      </Text>

      <Text>Нажатий: {count}</Text>

      <Pressable
        style={styles.button}
        onPress={() => setCount((value) => value + 1)}
      >
        <Text style={styles.buttonText}>
          Проверить кнопку
        </Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    padding: 24,
    gap: 20,
    backgroundColor: '#F3F5F8',
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: '#172033',
  },
  description: {
    fontSize: 18,
    color: '#526079',
  },
  button: {
    backgroundColor: '#2563EB',
    padding: 18,
    borderRadius: 12,
    alignItems: 'center',
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 18,
    fontWeight: '600',
  },
});
import { StyleSheet } from 'react-native';
export const taskStyles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#f5f6f8' },
  content: { padding: 20, gap: 16, width: '100%', maxWidth: 820, alignSelf: 'center' },
  title: { fontSize: 26, fontWeight: '700', color: '#182c43' },
  subtitle: { fontSize: 16, lineHeight: 23, color: '#59697d' },
  card: { padding: 18, gap: 10, borderWidth: 1, borderColor: '#d8dfe7', borderRadius: 10, backgroundColor: '#fff' },
  cardTitle: { fontSize: 20, fontWeight: '600', color: '#182c43' },
  text: { fontSize: 17, lineHeight: 24, color: '#28374b' },
  label: { fontSize: 14, color: '#59697d', marginBottom: 4 },
  field: { paddingVertical: 8, borderBottomWidth: StyleSheet.hairlineWidth, borderBottomColor: '#d8dfe7' },
  button: { minHeight: 50, paddingHorizontal: 18, paddingVertical: 14, alignItems: 'center', justifyContent: 'center', backgroundColor: '#235dae', borderRadius: 8 },
  secondary: { backgroundColor: '#e9eef5' },
  buttonText: { color: '#fff', fontSize: 17, fontWeight: '600', textAlign: 'center' },
  secondaryText: { color: '#28374b' },
  disabled: { opacity: 0.5 },
  message: { padding: 16, borderRadius: 8, backgroundColor: '#e5f2eb', color: '#23563d', fontSize: 16, lineHeight: 23 },
  error: { padding: 16, borderRadius: 8, backgroundColor: '#fce9e9', color: '#8d2828', fontSize: 16, lineHeight: 23 },
});

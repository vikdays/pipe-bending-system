import { Icon } from '../../components/Icon';
import { Modal } from '../../components/Modal';

export function ExchangeInfoModal({ onClose }: { onClose: () => void }) {
  return (
    <Modal
      title="Обмен с планшетом"
      description="Два автономных рабочих места и файлы на USB-накопителе."
      onClose={onClose}
    >
      <div className="exchange-steps">
        <div>
          <Icon name="upload" />
          <section>
            <h3>ПК → планшет</h3>
            <p>Задание, снимок сведений о трубе и версия конфигурации.</p>
          </section>
        </div>
        <div>
          <Icon name="download" />
          <section>
            <h3>Планшет → ПК</h3>
            <p>Подтверждённые измерения, результаты сеанса и PDF-протокол.</p>
          </section>
        </div>
        <div>
          <Icon name="shield" />
          <section>
            <h3>Базы остаются на устройствах</h3>
            <p>После импорта задания флешка для работы машиниста не нужна.</p>
          </section>
        </div>
      </div>
      <div className="inline-note">
        <Icon name="info" size={18} />
        <p>Файловый обмен будет подключён на следующем этапе разработки.</p>
      </div>
      <div className="modal-actions">
        <button className="button button--primary" onClick={onClose}>
          Понятно
        </button>
      </div>
    </Modal>
  );
}

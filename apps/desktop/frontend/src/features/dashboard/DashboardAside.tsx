import { Icon } from '../../components/Icon';

export function DashboardAside({
  onOpenExchange,
}: {
  onOpenExchange: () => void;
}) {
  return (
    <aside className="dashboard-aside" aria-label="Действия и порядок работы">
      <section className="exchange-card">
        <div className="exchange-symbol">
          <Icon name="exchange" size={27} />
        </div>
        <span className="eyebrow">ПК ↔ ПЛАНШЕТ</span>
        <h2>Обмен без Интернета</h2>
        <p>
          Задания — на планшет.
          <br />
          Результаты и протоколы — обратно на ПК.
        </p>
        <div className="usb-note">
          <Icon name="folder" size={16} />
          <span>Через USB-накопитель</span>
        </div>
        <button className="exchange-link" onClick={onOpenExchange}>
          Как устроен обмен
          <Icon name="arrow" size={17} />
        </button>
      </section>
      <section className="panel workflow-panel">
        <h2>Порядок работы</h2>
        <ol>
          <li>
            <span>1</span>
            <div>
              <strong>Подготовьте задание</strong>
              <p>Выберите трубу и параметры будущего отвода.</p>
            </div>
          </li>
          <li>
            <span>2</span>
            <div>
              <strong>Передайте машинисту</strong>
              <p>Экспортируйте файл задания на флешку.</p>
            </div>
          </li>
          <li>
            <span>3</span>
            <div>
              <strong>Получите результат</strong>
              <p>Импортируйте измерения и PDF-протокол.</p>
            </div>
          </li>
        </ol>
      </section>
      <div className="next-step-note">
        <Icon name="info" size={16} />
        <p>
          Трубный журнал, конфигурации и файловый обмен будут доступны на
          следующих этапах.
        </p>
      </div>
    </aside>
  );
}

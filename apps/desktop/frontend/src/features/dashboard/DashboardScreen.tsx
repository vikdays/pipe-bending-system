import { Icon } from '../../components/Icon';
import { Toast } from '../../components/Toast';
import { AppShell } from '../../components/layout/AppShell';
import { CreateTaskModal } from '../tasks/CreateTaskModal';
import { TaskDetailsModal } from '../tasks/TaskDetailsModal';
import { TasksPanel } from '../tasks/TasksPanel';
import { TaskSummaryCards } from '../tasks/TaskSummaryCards';
import { taskService } from '../tasks/taskClient';
import { pipeService } from '../pipes/pipeClient';
import { documentService } from '../pipes/documentClient';
import { PipesScreen } from '../pipes/PipesScreen';
import { DashboardAside } from './DashboardAside';
import { ExchangeInfoModal } from './ExchangeInfoModal';
import { useDashboard } from './useDashboard';
import type { AuthUser } from '../auth/authClientCore';

export function DashboardScreen({
  user,
  onOpenAccount,
  onSignOut,
}: {
  user: AuthUser;
  onOpenAccount: () => void;
  onSignOut: () => void;
}) {
  const dashboard = useDashboard(taskService, pipeService);

  return (
    <>
      <AppShell
        view={dashboard.view}
        taskCount={dashboard.tasks.length}
        onNavigate={dashboard.navigate}
        onSignOut={onSignOut}
        user={user}
        onOpenAccount={onOpenAccount}
      >
        {dashboard.view === 'pipes' ? (
          <PipesScreen
            service={pipeService}
            documents={documentService}
            onSaved={dashboard.retryLoading}
          />
        ) : (
          <>
            <div className="page-heading">
              <div>
                <p className="page-kicker">РАБОЧЕЕ МЕСТО РУКОВОДИТЕЛЯ</p>
                <h1>
                  {dashboard.view === 'overview'
                    ? 'Обзор работы'
                    : 'Задания на изготовление'}
                </h1>
                <p>
                  Подготовка заданий и контроль результатов изготовления
                  отводов.
                </p>
              </div>
              <div className="page-actions">
                <button
                  className="button button--secondary"
                  disabled={dashboard.isLoading}
                  onClick={dashboard.retryLoading}
                >
                  Обновить
                </button>
                <button
                  className="button button--primary"
                  onClick={dashboard.openCreate}
                  disabled={
                    dashboard.isLoading || dashboard.pipeOptions.length === 0
                  }
                >
                  <Icon name="plus" size={19} />
                  Создать задание
                </button>
              </div>
            </div>
            <div className="workspace-note">
              <Icon name="info" size={17} />
              <span>
                Трубы и черновики заданий сохраняются на этом компьютере.
                {dashboard.pipeOptions.length === 0 &&
                  ' Для первого задания добавьте трубу в трубный журнал.'}
              </span>
              {dashboard.pipeOptions.length === 0 && (
                <button
                  className="button button--secondary"
                  onClick={() => dashboard.navigate('pipes')}
                >
                  Открыть журнал
                </button>
              )}
            </div>
            <TaskSummaryCards
              total={dashboard.tasks.length}
              counts={dashboard.counts}
              onFilter={dashboard.applyFilter}
            />
            <div className="dashboard-grid">
              <TasksPanel
                tasks={dashboard.visibleTasks}
                total={dashboard.tasks.length}
                counts={dashboard.counts}
                query={dashboard.query}
                filter={dashboard.filter}
                isLoading={dashboard.isLoading}
                error={dashboard.loadError}
                onQueryChange={dashboard.setQuery}
                onFilterChange={dashboard.setFilter}
                onSelectTask={dashboard.openTask}
                onResetFilters={dashboard.resetFilters}
                onRetry={dashboard.retryLoading}
              />
              <DashboardAside onOpenExchange={dashboard.openExchange} />
            </div>
          </>
        )}
      </AppShell>
      {dashboard.dialog?.kind === 'create' && (
        <CreateTaskModal
          nextNumber={dashboard.tasks.length + 1}
          pipeOptions={dashboard.pipeOptions}
          isSubmitting={dashboard.isCreating}
          error={dashboard.createError}
          onCreate={dashboard.createTask}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'task' && (
        <TaskDetailsModal
          task={dashboard.dialog.task}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'exchange' && (
        <ExchangeInfoModal onClose={dashboard.closeDialog} />
      )}
      {dashboard.toast && (
        <Toast message={dashboard.toast} onDismiss={dashboard.dismissToast} />
      )}
    </>
  );
}

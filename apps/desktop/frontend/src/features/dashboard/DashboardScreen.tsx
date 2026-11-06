import { Icon } from '../../components/Icon';
import { Toast } from '../../components/Toast';
import { AppShell } from '../../components/layout/AppShell';
import { CreateTaskModal } from '../tasks/CreateTaskModal';
import { ExportTasksModal } from '../tasks/ExportTasksModal';
import { TaskDetailsModal } from '../tasks/TaskDetailsModal';
import { TasksPanel } from '../tasks/TasksPanel';
import { TaskSummaryCards } from '../tasks/TaskSummaryCards';
import { taskService } from '../tasks/taskClient';
import { pipeService } from '../pipes/pipeClient';
import { documentService } from '../pipes/documentClient';
import { PipesScreen } from '../pipes/PipesScreen';
import { DashboardAside } from './DashboardAside';
import { ExchangeInfoModal } from './ExchangeInfoModal';
import { TechnicalCardsModal } from '../tasks/TechnicalCardsModal';
import { technicalCardService } from '../tasks/technicalCardClient';
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
                <h1>
                  {dashboard.view === 'overview'
                    ? 'Обзор работы'
                    : 'Задания на изготовление'}
                </h1>
              </div>
              <div className="page-actions">
                <button
                  className="button button--secondary"
                  disabled={dashboard.isLoading || dashboard.counts.ready === 0}
                  onClick={dashboard.openExport}
                >
                  Выгрузить задания ({dashboard.counts.ready})
                </button>
                <button
                  className="button button--secondary"
                  disabled={dashboard.isLoading}
                  onClick={dashboard.openCards}
                >
                  Техкарты и конфигурации
                </button>
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
                  Создать задания
                </button>
              </div>
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
          task={dashboard.dialog.task}
          reviewConfiguration={dashboard.dialog.reviewConfiguration}
          pipeOptions={dashboard.pipeOptions}
          configurations={dashboard.configurations}
          machines={dashboard.machines}
          isSubmitting={dashboard.isCreating}
          error={dashboard.createError}
          onSave={dashboard.createTask}
          onConfigurationSaved={dashboard.saveConfiguration}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'task' && (
        <TaskDetailsModal
          task={dashboard.dialog.task}
          busy={dashboard.taskBusy}
          error={dashboard.taskError}
          onEdit={dashboard.editTask}
          onPrepare={dashboard.prepareTask}
          onReviewConfiguration={dashboard.reviewTaskConfiguration}
          onReload={dashboard.openTask}
          onOpenOriginal={technicalCardService.openOriginal}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'cards' && (
        <TechnicalCardsModal
          machines={dashboard.machines}
          configurations={dashboard.configurations}
          service={technicalCardService}
          onConfigurationSaved={dashboard.saveConfiguration}
          onImported={dashboard.refreshConfigurations}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'export' && (
        <ExportTasksModal
          service={taskService}
          onSaved={dashboard.exportSaved}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.dialog?.kind === 'exchange' && (
        <ExchangeInfoModal
          service={taskService}
          onClose={dashboard.closeDialog}
        />
      )}
      {dashboard.toast && (
        <Toast message={dashboard.toast} onDismiss={dashboard.dismissToast} />
      )}
    </>
  );
}

package mock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/domain"
)

// MockTaskRepo is a thread-safe in-memory task repository
type MockTaskRepo struct {
	mu           sync.Mutex
	tasks        map[int64]*domain.Task
	tasksByUUID  map[uuid.UUID]*domain.Task
	nextID       int64
	createdCount int64
}

func NewMockTaskRepo() *MockTaskRepo {
	return &MockTaskRepo{
		tasks:       make(map[int64]*domain.Task),
		tasksByUUID: make(map[uuid.UUID]*domain.Task),
		nextID:      1,
	}
}

func (m *MockTaskRepo) Create(ctx context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt64(&m.createdCount, 1)

	task.ID = m.nextID
	m.nextID++
	if task.UUID == uuid.Nil {
		task.UUID = uuid.New()
	}

	copied := *task
	m.tasks[task.ID] = &copied
	m.tasksByUUID[task.UUID] = &copied
	return nil
}

func (m *MockTaskRepo) CreatedCount() int64 {
	return atomic.LoadInt64(&m.createdCount)
}

func (m *MockTaskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok || task.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}
	copied := *task
	return &copied, nil
}

func (m *MockTaskRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasksByUUID[uid]
	if !ok || task.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}
	copied := *task
	return &copied, nil
}

func (m *MockTaskRepo) Update(ctx context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := *task
	m.tasks[task.ID] = &copied
	m.tasksByUUID[task.UUID] = &copied
	return nil
}

func (m *MockTaskRepo) Delete(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.tasks, id)
	delete(m.tasksByUUID, task.UUID)
	return nil
}

func (m *MockTaskRepo) List(ctx context.Context, query domain.ListTaskQuery) (*domain.TaskListResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []domain.Task
	for _, t := range m.tasks {
		if t.DeletedAt == nil && (t.CreatedBy == query.UserID || (t.AssigneeID != nil && *t.AssigneeID == query.UserID)) {
			list = append(list, *t)
		}
	}
	return &domain.TaskListResult{Items: list, TotalCount: int64(len(list))}, nil
}

// MockUserRepo
type MockUserRepo struct {
	mu     sync.Mutex
	users  map[int64]*domain.User
	byUUID map[uuid.UUID]*domain.User
	byMail map[string]*domain.User
	nextID int64
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users:  make(map[int64]*domain.User),
		byUUID: make(map[uuid.UUID]*domain.User),
		byMail: make(map[string]*domain.User),
		nextID: 1,
	}
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.ID = m.nextID
	m.nextID++
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}
	copied := *user
	m.users[user.ID] = &copied
	m.byUUID[user.UUID] = &copied
	m.byMail[user.Email] = &copied
	return nil
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *u
	return &copied, nil
}

func (m *MockUserRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.byUUID[uid]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *u
	return &copied, nil
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.byMail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *u
	return &copied, nil
}

// MockTeamRepo
type MockTeamRepo struct {
	mu      sync.Mutex
	teams   map[int64]*domain.Team
	members map[string]*domain.TeamMember
	nextID  int64
}

func NewMockTeamRepo() *MockTeamRepo {
	return &MockTeamRepo{
		teams:   make(map[int64]*domain.Team),
		members: make(map[string]*domain.TeamMember),
		nextID:  1,
	}
}

func (m *MockTeamRepo) Create(ctx context.Context, team *domain.Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team.ID = m.nextID
	m.nextID++
	copied := *team
	m.teams[team.ID] = &copied
	return nil
}

func (m *MockTeamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.teams[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *t
	return &copied, nil
}

func (m *MockTeamRepo) GetByUUID(ctx context.Context, uid uuid.UUID) (*domain.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, t := range m.teams {
		if t.UUID == uid {
			copied := *t
			return &copied, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockTeamRepo) AddMember(ctx context.Context, member *domain.TeamMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(rune(member.TeamID)) + ":" + string(rune(member.UserID))
	copied := *member
	m.members[key] = &copied
	return nil
}

func (m *MockTeamRepo) GetMember(ctx context.Context, teamID, userID int64) (*domain.TeamMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(rune(teamID)) + ":" + string(rune(userID))
	mem, ok := m.members[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copied := *mem
	return &copied, nil
}

func (m *MockTeamRepo) AreUsersInSameTeam(ctx context.Context, userID1, userID2 int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	teamsUser1 := make(map[int64]bool)
	for _, mem := range m.members {
		if mem.UserID == userID1 {
			teamsUser1[mem.TeamID] = true
		}
	}

	for _, mem := range m.members {
		if mem.UserID == userID2 && teamsUser1[mem.TeamID] {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockTeamRepo) ListUserTeams(ctx context.Context, userID int64) ([]domain.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var res []domain.Team
	for _, mem := range m.members {
		if mem.UserID == userID {
			if t, ok := m.teams[mem.TeamID]; ok {
				res = append(res, *t)
			}
		}
	}
	return res, nil
}

// MockTaskLogRepo
type MockTaskLogRepo struct {
	mu   sync.Mutex
	logs []domain.TaskLog
}

func NewMockTaskLogRepo() *MockTaskLogRepo {
	return &MockTaskLogRepo{
		logs: make([]domain.TaskLog, 0),
	}
}

func (m *MockTaskLogRepo) Create(ctx context.Context, log *domain.TaskLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logs = append(m.logs, *log)
	return nil
}

func (m *MockTaskLogRepo) ListByTaskID(ctx context.Context, taskID int64) ([]domain.TaskLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var res []domain.TaskLog
	for _, l := range m.logs {
		if l.TaskID == taskID {
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *MockTaskLogRepo) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs)
}

// MockTxManager simulating transaction and rollback
type MockTxManager struct {
	taskRepo    *MockTaskRepo
	taskLogRepo *MockTaskLogRepo
}

func NewMockTxManager(taskRepo *MockTaskRepo, logRepo *MockTaskLogRepo) *MockTxManager {
	return &MockTxManager{
		taskRepo:    taskRepo,
		taskLogRepo: logRepo,
	}
}

func (m *MockTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.taskRepo.mu.Lock()
	tasksSnapshot := make(map[int64]*domain.Task)
	tasksUUIDSnapshot := make(map[uuid.UUID]*domain.Task)
	for k, v := range m.taskRepo.tasks {
		copied := *v
		tasksSnapshot[k] = &copied
	}
	for k, v := range m.taskRepo.tasksByUUID {
		copied := *v
		tasksUUIDSnapshot[k] = &copied
	}
	m.taskRepo.mu.Unlock()

	m.taskLogRepo.mu.Lock()
	logsSnapshot := make([]domain.TaskLog, len(m.taskLogRepo.logs))
	copy(logsSnapshot, m.taskLogRepo.logs)
	m.taskLogRepo.mu.Unlock()

	err := fn(ctx)
	if err != nil {
		m.taskRepo.mu.Lock()
		m.taskRepo.tasks = tasksSnapshot
		m.taskRepo.tasksByUUID = tasksUUIDSnapshot
		m.taskRepo.mu.Unlock()

		m.taskLogRepo.mu.Lock()
		m.taskLogRepo.logs = logsSnapshot
		m.taskLogRepo.mu.Unlock()

		return err
	}

	return nil
}

// MockNotifier
type MockNotifier struct {
	mu        sync.Mutex
	ShouldErr bool
	SentCount int
}

func (n *MockNotifier) Send(ctx context.Context, recipientID int64, title, message string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.ShouldErr {
		return errors.New("notification service connection failed")
	}
	n.SentCount++
	return nil
}

// MockTaskCache
type MockTaskCache struct {
	mu           sync.Mutex
	items        map[uuid.UUID]*domain.TaskResponse
	deletedUUIDs []uuid.UUID
}

func NewMockTaskCache() *MockTaskCache {
	return &MockTaskCache{
		items:        make(map[uuid.UUID]*domain.TaskResponse),
		deletedUUIDs: make([]uuid.UUID, 0),
	}
}

func (m *MockTaskCache) Get(ctx context.Context, taskUUID uuid.UUID) (*domain.TaskResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.items[taskUUID]; ok {
		copied := *t
		return &copied, nil
	}
	return nil, nil
}

func (m *MockTaskCache) Set(ctx context.Context, task *domain.TaskResponse, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *task
	m.items[task.UUID] = &copied
	return nil
}

func (m *MockTaskCache) Delete(ctx context.Context, taskUUID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, taskUUID)
	m.deletedUUIDs = append(m.deletedUUIDs, taskUUID)
	return nil
}

func (m *MockTaskCache) DeletedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.deletedUUIDs)
}

package learnpath

import "testing"

// checkSequence 校验序列的通用合法性：不含已完成课程、无重复、
// 每门课的直接先修要么已完成要么出现在序列更前面。
func checkSequence(t *testing.T, catalog Catalog, seq []Course, completed []string) {
	t.Helper()
	done := map[string]bool{}
	for _, id := range completed {
		done[id] = true
	}
	byID := map[string]Course{}
	for _, c := range catalog.Courses {
		byID[c.ID] = c
	}
	seen := map[string]bool{}
	for _, c := range seq {
		if done[c.ID] {
			t.Errorf("序列包含已完成课程 %s", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("序列中课程 %s 重复出现", c.ID)
		}
		seen[c.ID] = true
		for _, p := range byID[c.ID].Prereqs {
			if !done[p] && !seen[p] {
				t.Errorf("课程 %s 的先修 %s 未完成且未排在更前面", c.ID, p)
			}
		}
	}
}

// checkPlan 校验序列与已完成课程共同满足计划的全部必修与选修要求。
func checkPlan(t *testing.T, plan Plan, seq []Course, completed []string) {
	t.Helper()
	counted := map[string]bool{}
	for _, id := range completed {
		counted[id] = true
	}
	for _, c := range seq {
		counted[c.ID] = true
	}
	for _, id := range plan.Required {
		if !counted[id] {
			t.Errorf("必修课 %s 未完成且不在序列中", id)
		}
	}
	for i, g := range plan.Groups {
		n := 0
		for _, id := range g.Candidates {
			if counted[id] {
				n++
			}
		}
		if n < g.Required {
			t.Errorf("选修组 %d 需 %d 门，实际仅 %d 门", i, g.Required, n)
		}
	}
}

func totalMinutes(seq []Course) int {
	total := 0
	for _, c := range seq {
		total += c.Minutes
	}
	return total
}

// TestSharedPrereqCountedOnce 验证共享先修只计费一次：
// 组1从 A、C 选一门，组2从 B、D 选一门。A、B 各 1 分钟，分别依赖
// 8 分钟的 P、Q；C、D 各 3 分钟，共同依赖 7 分钟的 R。
// 逐组贪心会选 A、B 得到 18 分钟，最优是 C、D 共享 R 共 13 分钟。
func TestSharedPrereqCountedOnce(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "P", Minutes: 8},
		{ID: "Q", Minutes: 8},
		{ID: "R", Minutes: 7},
		{ID: "A", Minutes: 1, Prereqs: []string{"P"}},
		{ID: "B", Minutes: 1, Prereqs: []string{"Q"}},
		{ID: "C", Minutes: 3, Prereqs: []string{"R"}},
		{ID: "D", Minutes: 3, Prereqs: []string{"R"}},
	}}
	plan := Plan{
		Groups: []ElectiveGroup{
			{Candidates: []string{"A", "C"}, Required: 1},
			{Candidates: []string{"B", "D"}, Required: 1},
		},
	}

	seq := Generate(catalog, plan, nil)

	checkSequence(t, catalog, seq, nil)
	checkPlan(t, plan, seq, nil)
	if got := totalMinutes(seq); got != 13 {
		t.Errorf("最优总时长应为 13 分钟（C、D 共享先修 R），实际 %d 分钟", got)
	}
}

// TestPrereqCountsForMultipleGroups 验证：作为先修进入序列的课程同时
// 贡献多个选修组名额；已完成课程历史先修缺失不会触发补修，且其
// 选修贡献不遗漏。
func TestPrereqCountsForMultipleGroups(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "X", Minutes: 5, Prereqs: []string{"Y"}},
		{ID: "Y", Minutes: 4},
		{ID: "Z", Minutes: 2, Prereqs: []string{"W"}},
		{ID: "W", Minutes: 6},
	}}
	plan := Plan{
		Required: []string{"X"},
		Groups: []ElectiveGroup{
			{Candidates: []string{"Y"}, Required: 1},
			{Candidates: []string{"Y"}, Required: 1},
			{Candidates: []string{"Z"}, Required: 1},
		},
	}
	completed := []string{"Z"} // Z 已完成，但其先修 W 未完成

	seq := Generate(catalog, plan, completed)

	checkSequence(t, catalog, seq, completed)
	checkPlan(t, plan, seq, completed)

	ids := map[string]bool{}
	for _, c := range seq {
		ids[c.ID] = true
	}
	if !ids["Y"] || !ids["X"] {
		t.Errorf("序列应包含 Y（X 的先修，同时满足两个选修组）和 X，实际 %v", ids)
	}
	if ids["W"] {
		t.Errorf("已完成课程 Z 的历史先修 W 不应被补修")
	}
	if ids["Z"] {
		t.Errorf("已完成课程 Z 不应出现在序列中")
	}
	if got := totalMinutes(seq); got != 9 {
		t.Errorf("总时长应为 9 分钟（Y=4 + X=5），实际 %d 分钟", got)
	}
	// Y 作为先修进入序列，须排在 X 之前且同时计入两个选修组：
	// 若 Y 未计入，两组各缺 1 门将无法由其他课程补足，checkPlan 已覆盖。
}

// TestAlreadySatisfiedReturnsEmpty 验证计划已被已完成课程满足时返回空序列。
func TestAlreadySatisfiedReturnsEmpty(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "A", Minutes: 3},
		{ID: "B", Minutes: 4},
	}}
	plan := Plan{
		Required: []string{"A"},
		Groups:   []ElectiveGroup{{Candidates: []string{"A", "B"}, Required: 1}},
	}

	seq := Generate(catalog, plan, []string{"A"})
	if len(seq) != 0 {
		t.Errorf("计划已满足，应返回空序列，实际 %v", seq)
	}
}

// TestRequiredWithPrereqChain 验证必修课的整条先修链都进入序列且顺序合法。
func TestRequiredWithPrereqChain(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "A", Minutes: 1, Prereqs: []string{"B"}},
		{ID: "B", Minutes: 2, Prereqs: []string{"C"}},
		{ID: "C", Minutes: 3},
	}}
	plan := Plan{Required: []string{"A"}}

	seq := Generate(catalog, plan, nil)

	checkSequence(t, catalog, seq, nil)
	checkPlan(t, plan, seq, nil)
	if got := totalMinutes(seq); got != 6 {
		t.Errorf("总时长应为 6 分钟（A+B+C），实际 %d 分钟", got)
	}
	if len(seq) != 3 || seq[0].ID != "C" || seq[1].ID != "B" || seq[2].ID != "A" {
		t.Errorf("序列顺序应为 C、B、A，实际 %v", seq)
	}
}

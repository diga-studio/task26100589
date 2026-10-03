package learnpath

import (
	"sort"
	"testing"
)

// validate 校验 seq 是否是一条满足全部约束的合法学习序列，
// 并返回序列的总时长（分钟）。
func validate(t *testing.T, catalog Catalog, plan Plan, completed []string, seq []string) int {
	t.Helper()
	courses := make(map[string]Course, len(catalog.Courses))
	for _, c := range catalog.Courses {
		courses[c.ID] = c
	}
	done := make(map[string]bool, len(completed))
	for _, id := range completed {
		done[id] = true
	}

	// 序列只含未完成的课程，每门最多一次，且先修已完成或排在更前面。
	seen := make(map[string]bool, len(seq))
	total := 0
	for i, id := range seq {
		c, ok := courses[id]
		if !ok {
			t.Fatalf("seq[%d]=%q 不在目录中", i, id)
		}
		if done[id] {
			t.Fatalf("seq[%d]=%q 已完成，不应出现在序列中", i, id)
		}
		if seen[id] {
			t.Fatalf("seq[%d]=%q 重复出现", i, id)
		}
		for _, p := range c.Prereqs {
			if !done[p] && !seen[p] {
				t.Fatalf("seq[%d]=%q 的先修 %q 未完成也未排在更前面", i, id, p)
			}
		}
		seen[id] = true
		total += c.Minutes
	}

	// 必修课必须已完成或在序列中。
	for _, id := range plan.Required {
		if !done[id] && !seen[id] {
			t.Fatalf("必修课 %q 未完成也未出现在序列中", id)
		}
	}

	// 每个选修组按已完成或在序列中的不同课程计数。
	for gi, g := range plan.Groups {
		count := 0
		for _, id := range g.Candidates {
			if done[id] || seen[id] {
				count++
			}
		}
		if count < g.MinCount {
			t.Fatalf("选修组 %d 需 %d 门，实际 %d 门", gi, g.MinCount, count)
		}
	}
	return total
}

func totalMinutes(catalog Catalog, seq []string) int {
	minutes := make(map[string]int, len(catalog.Courses))
	for _, c := range catalog.Courses {
		minutes[c.ID] = c.Minutes
	}
	total := 0
	for _, id := range seq {
		total += minutes[id]
	}
	return total
}

// 共享先修只计一次：两组分别选 C、D 时共享先修 R，
// 最优总时长 3+3+7=13，优于逐组独立选择得到的 1+1+8+8=18。
func TestGenerateSharedPrereqCountedOnce(t *testing.T) {
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
			{Candidates: []string{"A", "C"}, MinCount: 1},
			{Candidates: []string{"B", "D"}, MinCount: 1},
		},
	}

	seq := Generate(catalog, plan, nil)
	got := validate(t, catalog, plan, nil, seq)
	if got != 13 {
		t.Fatalf("最优总时长应为 13，实际 %d（序列 %v）", got, seq)
	}
	if totalMinutes(catalog, seq) != 13 {
		t.Fatalf("序列总时长应为 13，实际 %d", totalMinutes(catalog, seq))
	}
}

// 因先修依赖进入序列的课程同时贡献多个选修组；已完成课程的历史先修
// 不触发补修，且其选修贡献不被遗漏。
func TestGeneratePrereqContributesToGroupsAndCompletedTrusted(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "Core", Minutes: 5},
		{ID: "Adv", Minutes: 4, Prereqs: []string{"Core"}},
		{ID: "AltA", Minutes: 10},
		{ID: "AltB", Minutes: 10},
		{ID: "Legacy", Minutes: 3, Prereqs: []string{"Hist"}},
		{ID: "Hist", Minutes: 6},
		{ID: "Extra", Minutes: 9},
	}}
	plan := Plan{
		Required: []string{"Adv"},
		Groups: []ElectiveGroup{
			{Candidates: []string{"Core", "AltA"}, MinCount: 1},
			{Candidates: []string{"Core", "AltB"}, MinCount: 1},
			{Candidates: []string{"Legacy", "Extra"}, MinCount: 1},
		},
	}
	completed := []string{"Legacy"} // 历史先修 Hist 未完成，但可信

	seq := Generate(catalog, plan, completed)
	got := validate(t, catalog, plan, completed, seq)

	// Core 作为 Adv 的先修被拉入序列，同时满足前两个选修组；
	// Legacy 已完成，直接满足第三组，无需补修 Hist。
	want := []string{"Core", "Adv"}
	if len(seq) != len(want) {
		t.Fatalf("序列应为 %v，实际 %v", want, seq)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("序列应为 %v，实际 %v", want, seq)
		}
	}
	if got != 9 {
		t.Fatalf("总时长应为 9，实际 %d", got)
	}
	for _, id := range seq {
		if id == "Hist" {
			t.Fatalf("已完成课程的历史先修 Hist 不应被补修：%v", seq)
		}
	}
}

// 计划已被 completed 满足时返回空序列。
func TestGenerateAlreadySatisfied(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "X", Minutes: 5},
		{ID: "Y", Minutes: 5},
	}}
	plan := Plan{
		Required: []string{"X"},
		Groups:   []ElectiveGroup{{Candidates: []string{"X", "Y"}, MinCount: 1}},
	}
	if seq := Generate(catalog, plan, []string{"X"}); len(seq) != 0 {
		t.Fatalf("计划已满足，应返回空序列，实际 %v", seq)
	}
}

// 并列最优时任取其一，但必须合法且时长最小。
func TestGenerateTiedOptimumIsValid(t *testing.T) {
	catalog := Catalog{Courses: []Course{
		{ID: "M", Minutes: 2},
		{ID: "N", Minutes: 2},
	}}
	plan := Plan{
		Groups: []ElectiveGroup{{Candidates: []string{"M", "N"}, MinCount: 1}},
	}
	seq := Generate(catalog, plan, nil)
	if got := validate(t, catalog, plan, nil, seq); got != 2 {
		t.Fatalf("最优总时长应为 2，实际 %d（序列 %v）", got, seq)
	}
	sorted := append([]string(nil), seq...)
	sort.Strings(sorted)
	if len(sorted) != 1 || (sorted[0] != "M" && sorted[0] != "N") {
		t.Fatalf("并列最优应任选一门，实际 %v", seq)
	}
}

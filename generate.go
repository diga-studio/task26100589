package learnpath

import "sort"

// Generate 根据课程目录、学习计划与已完成课程集合，生成学习序列。
//
// 返回的序列只包含尚未完成的课程，每门最多出现一次；序列中任一课程的
// 每门直接先修课要么已完成，要么出现在序列更前面。序列满足全部必修与
// 选修要求，且在所有合法序列中新增学习总时长最小。并列最优时任取其一；
// 计划已被 completed 满足时返回空序列。
//
// completed 是可信的：已完成课程直接视为满足，即使其历史先修未完成也
// 不会触发补修。同一门课可同时贡献给多个选修组；因先修依赖被拉入序列
// 的课程同样贡献选修名额。
func Generate(catalog Catalog, plan Plan, completed []string) []string {
	s := newSolver(catalog, completed)
	for _, id := range plan.Required {
		s.expand(id)
	}
	for _, g := range plan.Groups {
		need := g.MinCount
		var cands []string
		for _, id := range g.Candidates {
			if s.completed[id] {
				need--
			} else {
				cands = append(cands, id)
			}
		}
		if need <= 0 {
			continue
		}
		if need > len(cands) {
			return nil // 该组无可行解，计划不可满足
		}
		s.groups = append(s.groups, cands)
		s.needs = append(s.needs, need)
	}
	s.best = -1
	s.search(0)
	if s.bestSet == nil {
		return nil
	}
	return topoSort(s.courses, s.bestSet)
}

// solver 在“每组选哪些选修课”的决策空间上做分支限界搜索。必修课及其
// 先修闭包在搜索前已固定纳入；每选择一个选修候选，就增量并入其先修闭包
// （跳过已完成课程），共享先修因去重只计费一次。
type solver struct {
	courses   map[string]Course
	completed map[string]bool
	included  map[string]bool
	cost      int
	groups    [][]string
	needs     []int
	best      int
	bestSet   map[string]bool
}

func newSolver(catalog Catalog, completed []string) *solver {
	s := &solver{
		courses:   make(map[string]Course, len(catalog.Courses)),
		completed: make(map[string]bool, len(completed)),
		included:  make(map[string]bool),
	}
	for _, c := range catalog.Courses {
		s.courses[c.ID] = c
	}
	for _, id := range completed {
		s.completed[id] = true
	}
	return s
}

// expand 将 id 及其缺失的先修闭包并入 included，返回本次新增的课程，
// 供回溯时撤销。已完成课程直接视为满足，不展开其历史先修。
func (s *solver) expand(id string) []string {
	var added []string
	stack := []string{id}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if s.completed[cur] || s.included[cur] {
			continue
		}
		s.included[cur] = true
		s.cost += s.courses[cur].Minutes
		added = append(added, cur)
		stack = append(stack, s.courses[cur].Prereqs...)
	}
	return added
}

func (s *solver) retract(added []string) {
	for _, id := range added {
		delete(s.included, id)
		s.cost -= s.courses[id].Minutes
	}
}

// search 递归处理第 gi 个未满足的选修组，枚举该组的所有候选组合。
func (s *solver) search(gi int) {
	if gi == len(s.groups) {
		if s.best < 0 || s.cost < s.best {
			s.best = s.cost
			s.bestSet = make(map[string]bool, len(s.included))
			for id := range s.included {
				s.bestSet[id] = true
			}
		}
		return
	}
	cands, need := s.groups[gi], s.needs[gi]
	var choose func(start, left int)
	choose = func(start, left int) {
		if left == 0 {
			if s.best < 0 || s.cost < s.best {
				s.search(gi + 1)
			}
			return
		}
		for i := start; i+left <= len(cands); i++ {
			added := s.expand(cands[i])
			choose(i+1, left-1)
			s.retract(added)
		}
	}
	choose(0, need)
}

// topoSort 对选中课程集合做拓扑排序，保证每门课的直接先修（若也在集合
// 中）排在它前面。就绪节点按 ID 字典序出队，使结果确定。
func topoSort(courses map[string]Course, set map[string]bool) []string {
	indeg := make(map[string]int, len(set))
	for id := range set {
		for _, p := range courses[id].Prereqs {
			if set[p] {
				indeg[id]++
			}
		}
	}
	var ready []string
	for id := range set {
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	var seq []string
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		seq = append(seq, id)
		for other, d := range indeg {
			if d <= 0 {
				continue
			}
			for _, p := range courses[other].Prereqs {
				if p == id {
					indeg[other]--
					if indeg[other] == 0 {
						ready = append(ready, other)
					}
				}
			}
		}
		sort.Strings(ready)
	}
	return seq
}

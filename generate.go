package learnpath

import "sort"

// Generate 根据课程目录、学习计划与已完成课程集合，生成学习序列。
//
// 返回的序列只包含尚未完成的课程，每门最多出现一次；序列中任一课程的
// 每门直接先修课要么已完成，要么出现在序列更前面。序列满足计划的全部
// 必修与选修要求，且序列中不同课程的分钟数之和在所有合法序列中最小。
// 计划已被 completed 满足时返回空序列。
func Generate(catalog Catalog, plan Plan, completed []string) []Course {
	byID := make(map[string]Course, len(catalog.Courses))
	for _, c := range catalog.Courses {
		byID[c.ID] = c
	}

	done := make(map[string]bool, len(completed))
	for _, id := range completed {
		done[id] = true
	}

	// closureOf 返回选择 id 后需要新增学习的全部课程（含 id 本身，
	// 不含已完成课程）。先修图无环，可安全记忆化。
	memo := make(map[string][]string)
	var closureOf func(id string) []string
	closureOf = func(id string) []string {
		if ids, ok := memo[id]; ok {
			return ids
		}
		set := map[string]bool{}
		if !done[id] {
			set[id] = true
			for _, p := range byID[id].Prereqs {
				for _, q := range closureOf(p) {
					set[q] = true
				}
			}
		}
		ids := make([]string, 0, len(set))
		for k := range set {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		memo[id] = ids
		return ids
	}

	// needed 是当前搜索状态下确定要学习的课程集合，cost 是其总分钟数。
	needed := map[string]bool{}
	cost := 0
	add := func(id string) {
		for _, q := range closureOf(id) {
			if !needed[q] {
				needed[q] = true
				cost += byID[q].Minutes
			}
		}
	}

	// 必修课必须全部进入序列。
	for _, id := range plan.Required {
		add(id)
	}

	// counted 判断课程是否计入某选修组（已完成或已在序列中）。
	counted := func(id string) bool {
		return done[id] || needed[id]
	}

	// 计算每个选修组仍缺的门数；已满足的组无需参与搜索。
	type groupState struct {
		choices []string // 尚未计入的候选课程
		deficit int      // 仍需选择的门数
	}
	var groups []groupState
	for _, g := range plan.Groups {
		gs := groupState{deficit: g.Required}
		for _, id := range g.Candidates {
			if counted(id) {
				gs.deficit--
			} else {
				gs.choices = append(gs.choices, id)
			}
		}
		if gs.deficit > 0 {
			groups = append(groups, gs)
		}
	}

	bestCost := -1
	var bestNeeded map[string]bool

	// 分支限界：依次为每个未满足的选修组选择 deficit 门候选课，
	// 共享先修通过 needed 集合只计费一次。
	var dfs func(gi int)
	var combine func(gi, start, left int)
	dfs = func(gi int) {
		if bestCost >= 0 && cost >= bestCost {
			return
		}
		if gi == len(groups) {
			bestCost = cost
			bestNeeded = make(map[string]bool, len(needed))
			for id := range needed {
				bestNeeded[id] = true
			}
			return
		}
		combine(gi, 0, groups[gi].deficit)
	}
	combine = func(gi, start, left int) {
		if left == 0 {
			dfs(gi + 1)
			return
		}
		gs := groups[gi]
		for i := start; i+left <= len(gs.choices); i++ {
			id := gs.choices[i]
			var added []string
			for _, q := range closureOf(id) {
				if !needed[q] {
					needed[q] = true
					cost += byID[q].Minutes
					added = append(added, q)
				}
			}
			combine(gi, i+1, left-1)
			for _, q := range added {
				delete(needed, q)
				cost -= byID[q].Minutes
			}
		}
	}
	dfs(0)

	return topoSort(bestNeeded, byID, catalog.Courses)
}

// topoSort 将选中的课程集合按先修依赖排成合法学习序列。
// 先修课排在后续课程之前；并列时按目录中的原始顺序，保证确定性。
func topoSort(set map[string]bool, byID map[string]Course, order []Course) []Course {
	if len(set) == 0 {
		return nil
	}
	indeg := make(map[string]int, len(set))
	for id := range set {
		for _, p := range byID[id].Prereqs {
			if set[p] {
				indeg[id]++
			}
		}
	}
	rank := make(map[string]int, len(order))
	for i, c := range order {
		rank[c.ID] = i
	}
	ready := make([]string, 0, len(set))
	for id := range set {
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return rank[ready[i]] < rank[ready[j]] })
	result := make([]Course, 0, len(set))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		result = append(result, byID[id])
		for _, c := range order {
			if !set[c.ID] {
				continue
			}
			for _, p := range c.Prereqs {
				if p == id {
					indeg[c.ID]--
					if indeg[c.ID] == 0 {
						ready = append(ready, c.ID)
						sort.Slice(ready, func(i, j int) bool { return rank[ready[i]] < rank[ready[j]] })
					}
				}
			}
		}
	}
	return result
}

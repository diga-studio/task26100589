// Package learnpath 提供学习路径编排能力：根据课程目录、学习计划与
// 已完成课程，生成满足全部必修、选修与先修要求且新增学习总时长最小
// 的学习序列。
package learnpath

// Course 描述目录中的一门课程。输入数据视为不可变。
type Course struct {
	// ID 是课程的唯一标识。
	ID string
	// Minutes 是课程时长，为正整数分钟数。
	Minutes int
	// Prereqs 是直接先修课程 ID 集合，必须全部满足。
	Prereqs []string
}

// Catalog 是课程目录，包含全部可选课程。
type Catalog struct {
	Courses []Course
}

// ElectiveGroup 描述一个选修组：从 Candidates 中至少完成 Required 门。
type ElectiveGroup struct {
	// Candidates 是该组候选课程 ID 集合。
	Candidates []string
	// Required 是至少需完成的门数，不超过候选数量。
	Required int
}

// Plan 是学习计划：必修课程集合加若干选修组。
type Plan struct {
	// Required 是必修课程 ID 集合。
	Required []string
	// Groups 是选修组列表；同一课程可属于多个组。
	Groups []ElectiveGroup
}

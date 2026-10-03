// Package learnpath 提供学习路径编排能力：根据课程目录、学习计划与
// 已完成课程集合，生成满足全部必修、选修与先修要求且新增学习总时长
// 最小的学习序列。
package learnpath

// Course 描述目录中的一门课程。
type Course struct {
	// ID 是课程的唯一标识。
	ID string
	// Minutes 是课程时长，单位分钟，正整数。
	Minutes int
	// Prereqs 是直接先修课程 ID 集合，全部满足后才可学习本课程。
	Prereqs []string
}

// Catalog 是内存中的课程目录，课程 ID 唯一，引用有效，先修图无环。
type Catalog struct {
	Courses []Course
}

// ElectiveGroup 描述一个选修组：候选课程中至少完成 MinCount 门。
type ElectiveGroup struct {
	// Candidates 是本组候选课程 ID 集合。
	Candidates []string
	// MinCount 是本组至少需完成的门数，不超过候选集合大小。
	MinCount int
}

// Plan 描述学习计划：必修课程集合与若干选修组。
type Plan struct {
	// Required 是必修课程 ID 集合。
	Required []string
	// Groups 是选修组列表。
	Groups []ElectiveGroup
}

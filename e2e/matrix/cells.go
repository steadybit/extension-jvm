//go:build matrix

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package matrix

import "fmt"

// Cell is one point in the support matrix: a sample built for a specific
// (Spring Boot, Java) combination. The build args mirror what the two sample
// Dockerfiles under testdata/samples expect.
type Cell struct {
	SampleType string // "spring" | "plain"
	Boot       string // Spring Boot version, "" for plain
	Java       string // Java LTS runtime, e.g. "21"
	Builder    string // build-stage image
	Runtime    string // run-stage image
	Compiler   string // maven.compiler release / javac --release
	RestClient bool   // sample exposes a RestClient endpoint (Boot >= 3.2); drives both the build profile and the attack set
}

func (c Cell) Name() string {
	if c.SampleType == "plain" {
		return fmt.Sprintf("plainjava-java%s", c.Java)
	}
	return fmt.Sprintf("boot%s-java%s", c.Boot, c.Java)
}

func (c Cell) Attacks() []AttackSpec {
	if c.SampleType == "plain" {
		return plainAttacks()
	}
	return springAttacks(c.RestClient)
}

// Cells is the sparse valid grid agreed for BM-13107. Illegal combinations
// (e.g. Boot 4.x on Java 8/11, Boot 2.7 on Java 21/25) are simply omitted.
func Cells() []Cell {
	var cells []Cell

	// Plain Java on every LTS (the two java-method attacks are Spring-independent).
	for _, j := range []string{"8", "11", "17", "21", "25"} {
		cells = append(cells, Cell{
			SampleType: "plain", Java: j, Compiler: "8",
			Builder: "eclipse-temurin:17-jdk", Runtime: "eclipse-temurin:" + j + "-jre",
		})
	}

	// Spring Boot 2.7 (Framework 5.x): Java 8-17. No RestClient.
	for _, j := range []string{"8", "11", "17"} {
		cells = append(cells, Cell{
			SampleType: "spring", Boot: "2.7.18", Java: j, Compiler: "8", RestClient: false,
			Builder: "maven:3.9.11-eclipse-temurin-8", Runtime: "eclipse-temurin:" + j + "-jre",
		})
	}

	// Spring Boot 3.5 (Framework 6.x): Java 17+.
	for _, j := range []string{"17", "21", "25"} {
		cells = append(cells, Cell{
			SampleType: "spring", Boot: "3.5.16", Java: j, Compiler: "17", RestClient: true,
			Builder: "maven:3.9.11-eclipse-temurin-17", Runtime: "eclipse-temurin:" + j + "-jre",
		})
	}

	// Spring Boot 4.1 (Framework 7.x): Java 17+.
	for _, j := range []string{"17", "21", "25"} {
		cells = append(cells, Cell{
			SampleType: "spring", Boot: "4.1.0", Java: j, Compiler: "17", RestClient: true,
			Builder: "maven:3.9.11-eclipse-temurin-21", Runtime: "eclipse-temurin:" + j + "-jre",
		})
	}

	return cells
}

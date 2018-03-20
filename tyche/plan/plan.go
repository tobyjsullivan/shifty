package plan

import "log"

type Step interface {
	Apply() error
	String() string
}

type Plan struct {
	Steps []Step
}

func (p *Plan) QueueStep(s Step) {
	p.Steps = append(p.Steps, s)
}

func (p *Plan) Apply() {
	for _, step := range p.Steps {
		log.Println("[Plan::Apply] Applying step: ", step.String())

		err := step.Apply()
		if err != nil {
			log.Fatalln("[Plan::Apply] Step failed:", err)
		} else {
			log.Println("[Plan::Apply] Step applied successfully.")
		}
	}
}

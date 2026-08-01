package agents

import "time"

// ProfessionCategory groups related professions so routines can share sensible
// defaults (typical work hours, common activity types).
type ProfessionCategory string

const (
	CategoryHealthcare  ProfessionCategory = "healthcare"
	CategoryEngineering ProfessionCategory = "engineering"
	CategoryTechnology  ProfessionCategory = "technology"
	CategoryEducation   ProfessionCategory = "education"
	CategoryLegal       ProfessionCategory = "legal"
	CategoryFinance     ProfessionCategory = "finance"
	CategoryBusiness    ProfessionCategory = "business"
	CategoryCreative    ProfessionCategory = "creative"
	CategoryTrades      ProfessionCategory = "trades"
	CategoryService     ProfessionCategory = "service"
	CategoryPublic      ProfessionCategory = "public_service"
	CategoryScience     ProfessionCategory = "science"
	CategoryAgriculture ProfessionCategory = "agriculture"
	CategorySports      ProfessionCategory = "sports"
)

// Activity is a unit of professional work an agent uses as a building block when
// generating a routine. Duration is the nominal length before intensity scaling.
type Activity struct {
	Name     string        `json:"name"`
	Category BlockCategory `json:"category"`
	Duration time.Duration `json:"duration"`
	Priority Priority      `json:"priority"`
}

// Profession describes a professional discipline and the raw material an agent
// uses to assemble a day: when the work day runs and which activities define it.
type Profession struct {
	Slug           string             `json:"slug"`
	Title          string             `json:"title"`
	Category       ProfessionCategory `json:"category"`
	Description    string             `json:"description"`
	WorkStart      string             `json:"work_start"` // "HH:MM", 24-hour
	WorkEnd        string             `json:"work_end"`   // "HH:MM", 24-hour
	CoreActivities []Activity         `json:"core_activities"`
}

func act(name string, cat BlockCategory, minutes int, prio Priority) Activity {
	return Activity{Name: name, Category: cat, Duration: time.Duration(minutes) * time.Minute, Priority: prio}
}

// DefaultProfessions returns the built-in catalog of "all professionals" that
// the routine generator can produce agents for. It intentionally spans every
// major category so DefaultRegistry can generate routines for the full set.
func DefaultProfessions() []Profession {
	return []Profession{
		// ---- Healthcare ----
		{
			Slug: "physician", Title: "Physician", Category: CategoryHealthcare,
			Description: "Diagnoses and treats patients in a clinical setting.",
			WorkStart:   "08:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Morning ward rounds", BlockClinical, 90, PriorityCritical),
				act("Patient consultations", BlockClinical, 120, PriorityHigh),
				act("Review labs and imaging", BlockAdmin, 45, PriorityHigh),
				act("Update patient charts", BlockAdmin, 45, PriorityMedium),
				act("Case discussion with team", BlockMeeting, 45, PriorityMedium),
			},
		},
		{
			Slug: "nurse", Title: "Registered Nurse", Category: CategoryHealthcare,
			Description: "Delivers and coordinates bedside patient care.",
			WorkStart:   "07:00", WorkEnd: "19:00",
			CoreActivities: []Activity{
				act("Shift handover", BlockMeeting, 30, PriorityHigh),
				act("Medication administration", BlockClinical, 60, PriorityCritical),
				act("Patient monitoring and vitals", BlockClinical, 120, PriorityHigh),
				act("Chart documentation", BlockAdmin, 60, PriorityMedium),
				act("Patient education", BlockClinical, 45, PriorityMedium),
			},
		},
		{
			Slug: "surgeon", Title: "Surgeon", Category: CategoryHealthcare,
			Description: "Performs operative procedures and pre/post-operative care.",
			WorkStart:   "06:30", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Pre-op review", BlockClinical, 45, PriorityCritical),
				act("Operating theatre", BlockClinical, 180, PriorityCritical),
				act("Post-op rounds", BlockClinical, 60, PriorityHigh),
				act("Surgical notes and dictation", BlockAdmin, 45, PriorityMedium),
			},
		},
		{
			Slug: "dentist", Title: "Dentist", Category: CategoryHealthcare,
			Description: "Provides dental examinations and treatments.",
			WorkStart:   "09:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Patient appointments", BlockClinical, 180, PriorityHigh),
				act("Treatment planning", BlockAdmin, 45, PriorityMedium),
				act("Sterilization and setup review", BlockAdmin, 30, PriorityMedium),
			},
		},
		{
			Slug: "pharmacist", Title: "Pharmacist", Category: CategoryHealthcare,
			Description: "Dispenses medication and advises on drug therapy.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Prescription verification", BlockClinical, 120, PriorityCritical),
				act("Patient counseling", BlockClinical, 60, PriorityHigh),
				act("Inventory and stock review", BlockAdmin, 45, PriorityMedium),
			},
		},
		{
			Slug: "veterinarian", Title: "Veterinarian", Category: CategoryHealthcare,
			Description: "Diagnoses and treats animal patients.",
			WorkStart:   "08:30", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Animal consultations", BlockClinical, 150, PriorityHigh),
				act("Procedures and surgery", BlockClinical, 90, PriorityHigh),
				act("Case records", BlockAdmin, 45, PriorityMedium),
			},
		},
		{
			Slug: "therapist", Title: "Physical Therapist", Category: CategoryHealthcare,
			Description: "Rehabilitates patients through movement and exercise.",
			WorkStart:   "08:00", WorkEnd: "16:30",
			CoreActivities: []Activity{
				act("Patient rehab sessions", BlockClinical, 180, PriorityHigh),
				act("Progress assessments", BlockAdmin, 60, PriorityMedium),
				act("Treatment plan updates", BlockAdmin, 45, PriorityMedium),
			},
		},

		// ---- Engineering ----
		{
			Slug: "civil-engineer", Title: "Civil Engineer", Category: CategoryEngineering,
			Description: "Designs and oversees infrastructure and construction.",
			WorkStart:   "08:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Site inspection", BlockFieldwork, 120, PriorityHigh),
				act("Structural design and calculations", BlockDeepWork, 120, PriorityHigh),
				act("Drawing and plan review", BlockDeepWork, 60, PriorityMedium),
				act("Contractor coordination", BlockMeeting, 45, PriorityMedium),
			},
		},
		{
			Slug: "mechanical-engineer", Title: "Mechanical Engineer", Category: CategoryEngineering,
			Description: "Designs mechanical systems and components.",
			WorkStart:   "08:30", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("CAD modeling", BlockDeepWork, 150, PriorityHigh),
				act("Prototype testing", BlockFieldwork, 90, PriorityHigh),
				act("Design review", BlockMeeting, 60, PriorityMedium),
				act("Documentation", BlockAdmin, 45, PriorityLow),
			},
		},
		{
			Slug: "electrical-engineer", Title: "Electrical Engineer", Category: CategoryEngineering,
			Description: "Designs and tests electrical systems.",
			WorkStart:   "08:30", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Circuit design", BlockDeepWork, 150, PriorityHigh),
				act("Lab testing and measurement", BlockFieldwork, 90, PriorityHigh),
				act("Schematic review", BlockMeeting, 45, PriorityMedium),
			},
		},

		// ---- Technology ----
		{
			Slug: "software-engineer", Title: "Software Engineer", Category: CategoryTechnology,
			Description: "Designs, builds, and maintains software systems.",
			WorkStart:   "09:00", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Daily standup", BlockMeeting, 15, PriorityMedium),
				act("Feature development", BlockDeepWork, 180, PriorityHigh),
				act("Code review", BlockDeepWork, 60, PriorityMedium),
				act("Bug triage and fixes", BlockDeepWork, 60, PriorityHigh),
				act("Technical learning", BlockLearning, 45, PriorityLow),
			},
		},
		{
			Slug: "data-scientist", Title: "Data Scientist", Category: CategoryTechnology,
			Description: "Extracts insight from data through modeling and analysis.",
			WorkStart:   "09:00", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Data exploration", BlockDeepWork, 120, PriorityHigh),
				act("Model training and evaluation", BlockDeepWork, 120, PriorityHigh),
				act("Stakeholder review", BlockMeeting, 45, PriorityMedium),
				act("Findings write-up", BlockAdmin, 60, PriorityMedium),
			},
		},
		{
			Slug: "devops-engineer", Title: "DevOps Engineer", Category: CategoryTechnology,
			Description: "Builds and operates deployment and infrastructure pipelines.",
			WorkStart:   "09:00", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Incident and alert review", BlockAdmin, 45, PriorityCritical),
				act("Pipeline and infra work", BlockDeepWork, 150, PriorityHigh),
				act("On-call handover", BlockMeeting, 30, PriorityMedium),
				act("Automation improvements", BlockDeepWork, 90, PriorityMedium),
			},
		},
		{
			Slug: "product-manager", Title: "Product Manager", Category: CategoryTechnology,
			Description: "Defines product direction and coordinates delivery.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Roadmap and backlog grooming", BlockDeepWork, 90, PriorityHigh),
				act("Stakeholder meetings", BlockMeeting, 120, PriorityHigh),
				act("User and data review", BlockDeepWork, 60, PriorityMedium),
				act("Spec writing", BlockDeepWork, 60, PriorityMedium),
			},
		},

		// ---- Education ----
		{
			Slug: "teacher", Title: "Teacher", Category: CategoryEducation,
			Description: "Instructs and mentors students in a school setting.",
			WorkStart:   "07:30", WorkEnd: "16:00",
			CoreActivities: []Activity{
				act("Lesson preparation", BlockDeepWork, 60, PriorityHigh),
				act("Classroom teaching", BlockClinical, 240, PriorityCritical),
				act("Grading and feedback", BlockAdmin, 60, PriorityMedium),
				act("Parent and staff communication", BlockMeeting, 30, PriorityMedium),
			},
		},
		{
			Slug: "professor", Title: "University Professor", Category: CategoryEducation,
			Description: "Teaches and conducts academic research.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Lectures", BlockClinical, 120, PriorityHigh),
				act("Research and writing", BlockDeepWork, 150, PriorityHigh),
				act("Office hours", BlockMeeting, 60, PriorityMedium),
				act("Grant and admin work", BlockAdmin, 45, PriorityLow),
			},
		},

		// ---- Legal ----
		{
			Slug: "lawyer", Title: "Lawyer", Category: CategoryLegal,
			Description: "Advises clients and represents them in legal matters.",
			WorkStart:   "09:00", WorkEnd: "18:30",
			CoreActivities: []Activity{
				act("Case research", BlockDeepWork, 120, PriorityHigh),
				act("Drafting documents", BlockDeepWork, 120, PriorityHigh),
				act("Client meetings", BlockMeeting, 90, PriorityHigh),
				act("Court or negotiation prep", BlockDeepWork, 60, PriorityMedium),
			},
		},

		// ---- Finance ----
		{
			Slug: "accountant", Title: "Accountant", Category: CategoryFinance,
			Description: "Maintains financial records and prepares reports.",
			WorkStart:   "09:00", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Ledger reconciliation", BlockDeepWork, 120, PriorityHigh),
				act("Financial reporting", BlockDeepWork, 90, PriorityHigh),
				act("Client or team review", BlockMeeting, 45, PriorityMedium),
				act("Compliance checks", BlockAdmin, 60, PriorityMedium),
			},
		},
		{
			Slug: "financial-analyst", Title: "Financial Analyst", Category: CategoryFinance,
			Description: "Analyzes financial data to guide investment decisions.",
			WorkStart:   "08:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Market review", BlockAdmin, 45, PriorityHigh),
				act("Financial modeling", BlockDeepWork, 150, PriorityHigh),
				act("Report preparation", BlockDeepWork, 90, PriorityMedium),
				act("Investment committee", BlockMeeting, 60, PriorityMedium),
			},
		},

		// ---- Business ----
		{
			Slug: "marketing-manager", Title: "Marketing Manager", Category: CategoryBusiness,
			Description: "Plans and executes marketing strategy and campaigns.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Campaign planning", BlockDeepWork, 90, PriorityHigh),
				act("Content and creative review", BlockCreative, 60, PriorityMedium),
				act("Analytics review", BlockAdmin, 45, PriorityMedium),
				act("Team and stakeholder syncs", BlockMeeting, 90, PriorityMedium),
			},
		},
		{
			Slug: "sales-representative", Title: "Sales Representative", Category: CategoryBusiness,
			Description: "Builds client relationships and closes deals.",
			WorkStart:   "08:30", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Pipeline review", BlockAdmin, 30, PriorityHigh),
				act("Prospecting and outreach", BlockDeepWork, 90, PriorityHigh),
				act("Client calls and demos", BlockMeeting, 150, PriorityCritical),
				act("CRM updates", BlockAdmin, 45, PriorityLow),
			},
		},
		{
			Slug: "entrepreneur", Title: "Entrepreneur", Category: CategoryBusiness,
			Description: "Builds and grows a business across many fronts.",
			WorkStart:   "08:00", WorkEnd: "19:00",
			CoreActivities: []Activity{
				act("Strategic planning", BlockDeepWork, 90, PriorityHigh),
				act("Investor and partner meetings", BlockMeeting, 90, PriorityHigh),
				act("Product and operations", BlockDeepWork, 120, PriorityHigh),
				act("Team leadership", BlockMeeting, 60, PriorityMedium),
			},
		},
		{
			Slug: "project-manager", Title: "Project Manager", Category: CategoryBusiness,
			Description: "Plans, tracks, and delivers projects on schedule.",
			WorkStart:   "09:00", WorkEnd: "17:30",
			CoreActivities: []Activity{
				act("Status and risk review", BlockAdmin, 45, PriorityHigh),
				act("Team stand-ups and syncs", BlockMeeting, 90, PriorityHigh),
				act("Planning and scheduling", BlockDeepWork, 90, PriorityMedium),
				act("Stakeholder reporting", BlockAdmin, 45, PriorityMedium),
			},
		},

		// ---- Creative ----
		{
			Slug: "graphic-designer", Title: "Graphic Designer", Category: CategoryCreative,
			Description: "Creates visual designs for brands and products.",
			WorkStart:   "09:30", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Design concepting", BlockCreative, 120, PriorityHigh),
				act("Production and revisions", BlockCreative, 120, PriorityHigh),
				act("Client feedback review", BlockMeeting, 45, PriorityMedium),
				act("Asset organization", BlockAdmin, 30, PriorityLow),
			},
		},
		{
			Slug: "writer", Title: "Writer", Category: CategoryCreative,
			Description: "Produces written content and long-form work.",
			WorkStart:   "08:00", WorkEnd: "15:00",
			CoreActivities: []Activity{
				act("Focused writing", BlockCreative, 180, PriorityCritical),
				act("Research", BlockDeepWork, 60, PriorityMedium),
				act("Editing and revision", BlockCreative, 90, PriorityHigh),
			},
		},
		{
			Slug: "photographer", Title: "Photographer", Category: CategoryCreative,
			Description: "Captures and produces photographic work.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Shoot preparation", BlockAdmin, 45, PriorityMedium),
				act("Photo session", BlockFieldwork, 180, PriorityHigh),
				act("Editing and retouching", BlockCreative, 120, PriorityHigh),
			},
		},

		// ---- Trades ----
		{
			Slug: "electrician", Title: "Electrician", Category: CategoryTrades,
			Description: "Installs and repairs electrical systems on site.",
			WorkStart:   "07:00", WorkEnd: "16:00",
			CoreActivities: []Activity{
				act("Job site setup", BlockFieldwork, 45, PriorityMedium),
				act("Installation and wiring", BlockFieldwork, 180, PriorityHigh),
				act("Testing and inspection", BlockFieldwork, 90, PriorityHigh),
				act("Job paperwork", BlockAdmin, 30, PriorityLow),
			},
		},
		{
			Slug: "plumber", Title: "Plumber", Category: CategoryTrades,
			Description: "Installs and repairs plumbing systems.",
			WorkStart:   "07:00", WorkEnd: "16:00",
			CoreActivities: []Activity{
				act("Route and job planning", BlockAdmin, 30, PriorityMedium),
				act("Repairs and installations", BlockFieldwork, 210, PriorityHigh),
				act("Customer walkthrough", BlockMeeting, 30, PriorityMedium),
			},
		},
		{
			Slug: "carpenter", Title: "Carpenter", Category: CategoryTrades,
			Description: "Builds and repairs wooden structures and fittings.",
			WorkStart:   "07:00", WorkEnd: "16:00",
			CoreActivities: []Activity{
				act("Material prep", BlockFieldwork, 45, PriorityMedium),
				act("Construction and assembly", BlockFieldwork, 210, PriorityHigh),
				act("Finishing work", BlockFieldwork, 60, PriorityMedium),
			},
		},
		{
			Slug: "mechanic", Title: "Automotive Mechanic", Category: CategoryTrades,
			Description: "Diagnoses and repairs vehicles.",
			WorkStart:   "08:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Vehicle diagnostics", BlockFieldwork, 90, PriorityHigh),
				act("Repairs and servicing", BlockFieldwork, 180, PriorityHigh),
				act("Customer updates", BlockMeeting, 30, PriorityMedium),
				act("Work orders", BlockAdmin, 30, PriorityLow),
			},
		},
		{
			Slug: "chef", Title: "Chef", Category: CategoryService,
			Description: "Leads kitchen operations and food preparation.",
			WorkStart:   "10:00", WorkEnd: "22:00",
			CoreActivities: []Activity{
				act("Menu and prep planning", BlockDeepWork, 60, PriorityHigh),
				act("Mise en place", BlockFieldwork, 90, PriorityHigh),
				act("Service line", BlockClinical, 240, PriorityCritical),
				act("Inventory and ordering", BlockAdmin, 45, PriorityMedium),
			},
		},

		// ---- Public service ----
		{
			Slug: "police-officer", Title: "Police Officer", Category: CategoryPublic,
			Description: "Maintains public safety and enforces the law.",
			WorkStart:   "07:00", WorkEnd: "19:00",
			CoreActivities: []Activity{
				act("Briefing and equipment check", BlockMeeting, 30, PriorityHigh),
				act("Patrol", BlockFieldwork, 240, PriorityHigh),
				act("Incident response", BlockFieldwork, 120, PriorityCritical),
				act("Reports and paperwork", BlockAdmin, 60, PriorityMedium),
			},
		},
		{
			Slug: "firefighter", Title: "Firefighter", Category: CategoryPublic,
			Description: "Responds to fires and emergencies.",
			WorkStart:   "08:00", WorkEnd: "08:00", // 24h shift
			CoreActivities: []Activity{
				act("Equipment and vehicle check", BlockFieldwork, 60, PriorityHigh),
				act("Drills and training", BlockLearning, 120, PriorityMedium),
				act("Emergency response", BlockFieldwork, 120, PriorityCritical),
				act("Station duties", BlockAdmin, 60, PriorityLow),
			},
		},

		// ---- Science ----
		{
			Slug: "research-scientist", Title: "Research Scientist", Category: CategoryScience,
			Description: "Conducts experiments and advances knowledge.",
			WorkStart:   "09:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Literature review", BlockLearning, 60, PriorityMedium),
				act("Lab experiments", BlockFieldwork, 180, PriorityHigh),
				act("Data analysis", BlockDeepWork, 90, PriorityHigh),
				act("Paper writing", BlockDeepWork, 60, PriorityMedium),
			},
		},

		// ---- Agriculture ----
		{
			Slug: "farmer", Title: "Farmer", Category: CategoryAgriculture,
			Description: "Manages crops, livestock, and land.",
			WorkStart:   "05:30", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Livestock and field check", BlockFieldwork, 120, PriorityCritical),
				act("Planting or harvesting", BlockFieldwork, 240, PriorityHigh),
				act("Equipment maintenance", BlockFieldwork, 60, PriorityMedium),
				act("Records and market planning", BlockAdmin, 45, PriorityLow),
			},
		},

		// ---- Sports ----
		{
			Slug: "athlete", Title: "Professional Athlete", Category: CategorySports,
			Description: "Trains and competes at a professional level.",
			WorkStart:   "07:00", WorkEnd: "17:00",
			CoreActivities: []Activity{
				act("Strength and conditioning", BlockWellness, 90, PriorityHigh),
				act("Skills and tactical training", BlockFieldwork, 120, PriorityCritical),
				act("Recovery and physiotherapy", BlockWellness, 60, PriorityHigh),
				act("Film and strategy review", BlockLearning, 60, PriorityMedium),
			},
		},
		{
			Slug: "personal-trainer", Title: "Personal Trainer", Category: CategorySports,
			Description: "Coaches clients toward fitness goals.",
			WorkStart:   "06:00", WorkEnd: "20:00",
			CoreActivities: []Activity{
				act("Client training sessions", BlockClinical, 240, PriorityHigh),
				act("Program design", BlockDeepWork, 60, PriorityMedium),
				act("Own workout", BlockWellness, 60, PriorityMedium),
				act("Client check-ins", BlockAdmin, 45, PriorityLow),
			},
		},
		{
			Slug: "pilot", Title: "Airline Pilot", Category: CategoryService,
			Description: "Operates aircraft and ensures flight safety.",
			WorkStart:   "06:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("Pre-flight briefing", BlockMeeting, 45, PriorityCritical),
				act("Flight operations", BlockFieldwork, 300, PriorityCritical),
				act("Post-flight checks and logs", BlockAdmin, 45, PriorityHigh),
			},
		},
		{
			Slug: "journalist", Title: "Journalist", Category: CategoryCreative,
			Description: "Researches, reports, and writes news stories.",
			WorkStart:   "08:00", WorkEnd: "18:00",
			CoreActivities: []Activity{
				act("News scan and pitch", BlockAdmin, 45, PriorityHigh),
				act("Interviews and reporting", BlockFieldwork, 120, PriorityHigh),
				act("Story writing", BlockCreative, 120, PriorityCritical),
				act("Editorial review", BlockMeeting, 45, PriorityMedium),
			},
		},
	}
}

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type AdvancedSeedConfig struct {
	NumClinics       int
	NumFormsPerClinic int
	NumFieldsPerForm int
	CreatePractitioner bool
	PractitionerEmail string
	Verbose          bool
}

func main() {
	// Command line flags
	numClinics := flag.Int("clinics", 10, "Number of clinics to create")
	numForms := flag.Int("forms", 5, "Number of forms per clinic")
	numFields := flag.Int("fields", 6, "Number of fields per form")
	createPractitioner := flag.Bool("create-practitioner", false, "Create a new practitioner instead of using existing")
	practitionerEmail := flag.String("practitioner-email", "", "Email for new practitioner (if creating)")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to database
	db, err := connectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Seed configuration
	config := AdvancedSeedConfig{
		NumClinics:        *numClinics,
		NumFormsPerClinic: *numForms,
		NumFieldsPerForm:  *numFields,
		CreatePractitioner: *createPractitioner,
		PractitionerEmail: *practitionerEmail,
		Verbose:           *verbose,
	}

	// Get or create a practitioner for seeding
	practitionerID, err := getOrCreatePractitionerAdvanced(db, config)
	if err != nil {
		log.Fatalf("Failed to get/create practitioner: %v", err)
	}

	log.Printf("Starting seed with practitioner ID: %s", practitionerID)
	log.Printf("Configuration: %d clinics, %d forms per clinic, %d fields per form", 
		config.NumClinics, config.NumFormsPerClinic, config.NumFieldsPerForm)

	// Seed clinics and forms
	if err := seedClinicsAndFormsAdvanced(db, practitionerID, config); err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	log.Println("✓ Seeding completed successfully!")
	log.Printf("Created %d clinics with %d forms each", config.NumClinics, config.NumFormsPerClinic)
}

func connectDB() (*sqlx.DB, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getOrCreatePractitionerAdvanced(db *sqlx.DB, config AdvancedSeedConfig) (uuid.UUID, error) {
	if config.CreatePractitioner {
		return createNewPractitioner(db, config.PractitionerEmail)
	}

	var practitionerID uuid.UUID
	err := db.Get(&practitionerID, "SELECT id FROM tbl_practitioner LIMIT 1")
	if err == nil {
		return practitionerID, nil
	}

	if err == sql.ErrNoRows {
		log.Println("No existing practitioner found, creating one...")
		return createNewPractitioner(db, config.PractitionerEmail)
	}

	return uuid.Nil, err
}

func createNewPractitioner(db *sqlx.DB, email string) (uuid.UUID, error) {
	userID := uuid.New()
	
	if email == "" {
		email = gofakeit.Email()
	}
	
	firstName := gofakeit.FirstName()
	lastName := gofakeit.LastName()
	
	_, err := db.Exec(`
		INSERT INTO tbl_user (id, email, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, email, firstName, lastName, "PRACTITIONER", true, time.Now(), time.Now())
	
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}

	practitionerID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO tbl_practitioner (id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`, practitionerID, userID, time.Now(), time.Now())
	
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create practitioner: %w", err)
	}

	log.Printf("Created new practitioner: %s (%s %s)", email, firstName, lastName)
	return practitionerID, nil
}

func seedClinicsAndFormsAdvanced(db *sqlx.DB, practitionerID uuid.UUID, config AdvancedSeedConfig) error {
	ctx := context.Background()
	startTime := time.Now()

	for i := 0; i < config.NumClinics; i++ {
		clinicID, clinicName, err := createClinicAdvanced(ctx, db, practitionerID, config)
		if err != nil {
			return fmt.Errorf("failed to create clinic %d: %w", i+1, err)
		}

		if config.Verbose {
			log.Printf("[%d/%d] Created clinic: %s (ID: %s)", i+1, config.NumClinics, clinicName, clinicID)
		} else {
			log.Printf("Created clinic %d/%d", i+1, config.NumClinics)
		}

		// Create forms for this clinic
		for j := 0; j < config.NumFormsPerClinic; j++ {
			formID, formName, err := createFormAdvanced(ctx, db, clinicID, config)
			if err != nil {
				return fmt.Errorf("failed to create form %d for clinic %s: %w", j+1, clinicID, err)
			}
			
			if config.Verbose {
				log.Printf("  [%d/%d] Created form: %s (ID: %s)", j+1, config.NumFormsPerClinic, formName, formID)
			}
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("Seeding completed in %s", elapsed)
	
	return nil
}

func createClinicAdvanced(ctx context.Context, db *sqlx.DB, practitionerID uuid.UUID, config AdvancedSeedConfig) (uuid.UUID, string, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, "", err
	}
	defer tx.Rollback()

	clinicID := uuid.New()
	entityID := uuid.New()
	
	// Generate realistic clinic data
	name := fmt.Sprintf("%s %s", gofakeit.Company(), gofakeit.RandomString([]string{"Clinic", "Medical Center", "Health Services", "Practice"}))
	abn := gofakeit.Numerify("###########")
	description := gofakeit.Sentence(gofakeit.Number(8, 15))
	
	// Insert clinic
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tbl_clinic (id, practitioner_id, entity_id, name, abn, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, clinicID, practitionerID, entityID, name, abn, description, true, time.Now(), time.Now())
	
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to insert clinic: %w", err)
	}

	// Insert multiple addresses (1-3 per clinic)
	numAddresses := gofakeit.Number(1, 3)
	for i := 0; i < numAddresses; i++ {
		addressID := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO tbl_clinic_address (id, clinic_id, address, city, state, postcode, is_primary, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, addressID, clinicID, gofakeit.Street(), gofakeit.City(), gofakeit.StateAbr(), 
			gofakeit.Numerify("####"), i == 0, time.Now(), time.Now())
		
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to insert clinic address: %w", err)
		}
	}

	// Insert multiple contacts (2-4 per clinic)
	contactTypes := []string{"EMAIL", "PHONE", "WEBSITE", "FAX"}
	numContacts := gofakeit.Number(2, 4)
	
	for i := 0; i < numContacts; i++ {
		contactID := uuid.New()
		contactType := contactTypes[i%len(contactTypes)]
		
		var value, label string
		switch contactType {
		case "EMAIL":
			value = gofakeit.Email()
			label = gofakeit.RandomString([]string{"Primary Email", "Admin Email", "Billing Email"})
		case "PHONE":
			value = gofakeit.Phone()
			label = gofakeit.RandomString([]string{"Main Phone", "Reception", "Emergency"})
		case "WEBSITE":
			value = fmt.Sprintf("https://%s.com", gofakeit.Username())
			label = "Website"
		case "FAX":
			value = gofakeit.Phone()
			label = "Fax"
		}
		
		_, err = tx.ExecContext(ctx, `
			INSERT INTO tbl_clinic_contact (id, clinic_id, contact_type, value, label, is_primary, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, contactID, clinicID, contactType, value, label, i == 0, time.Now(), time.Now())
		
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to insert clinic contact: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, "", err
	}

	return clinicID, name, nil
}

func createFormAdvanced(ctx context.Context, db *sqlx.DB, clinicID uuid.UUID, config AdvancedSeedConfig) (uuid.UUID, string, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, "", err
	}
	defer tx.Rollback()

	formID := uuid.New()
	
	// Generate realistic form names
	formTypes := []string{"Consultation", "Treatment", "Assessment", "Billing", "Service", "Procedure"}
	formName := fmt.Sprintf("%s %s Form", gofakeit.JobTitle(), gofakeit.RandomString(formTypes))
	description := gofakeit.Sentence(gofakeit.Number(6, 12))
	
	// Random method and status with weighted distribution
	methods := []string{"INDEPENDENT_CONTRACTOR", "SERVICE_FEE"}
	statuses := []string{"DRAFT", "PUBLISHED"}
	method := methods[gofakeit.Number(0, 1)]
	status := statuses[gofakeit.Number(0, 1)]
	
	// More realistic share distribution
	ownerShare := gofakeit.Number(40, 80)
	clinicShare := 100 - ownerShare
	superComponent := float64(gofakeit.Number(9, 12)) + gofakeit.Float64Range(0, 0.5)
	
	// Insert form
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tbl_form (id, clinic_id, name, description, status, method, owner_share, clinic_share, super_component, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, formID, clinicID, formName, description, status, method, ownerShare, clinicShare, superComponent, time.Now(), time.Now())
	
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to insert form: %w", err)
	}

	// Get practitioner ID from clinic
	var practitionerID uuid.UUID
	err = tx.GetContext(ctx, &practitionerID, `
		SELECT practitioner_id FROM tbl_clinic WHERE id = $1
	`, clinicID)
	
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get practitioner ID: %w", err)
	}

	// Create a form version
	versionID := uuid.New()
	versionNumber := 1
	
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tbl_custom_form_version (id, form_id, version, is_active, practitioner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, versionID, formID, versionNumber, true, practitionerID, time.Now(), time.Now())
	
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to insert form version: %w", err)
	}

	// Get or create COA
	var coaID uuid.UUID
	err = tx.GetContext(ctx, &coaID, `
		SELECT id FROM tbl_chart_of_accounts WHERE practitioner_id = $1 LIMIT 1
	`, practitionerID)
	
	if err == sql.ErrNoRows {
		coaID, err = createBasicCOAAdvanced(ctx, tx, practitionerID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to create COA: %w", err)
		}
	} else if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get COA: %w", err)
	}

	// Create form fields based on config
	fieldTemplates := []struct {
		key         string
		label       string
		isComputed  bool
		sectionType string
		taxType     string
	}{
		{"A", "Total Revenue", false, "COLLECTION", "INCLUSIVE"},
		{"B", "Service Fees", false, "COLLECTION", "EXCLUSIVE"},
		{"C", "Direct Costs", false, "COST", "EXCLUSIVE"},
		{"D", "Labor Costs", false, "COST", "INCLUSIVE"},
		{"E", "Operating Expenses", false, "OTHER_COST", "EXCLUSIVE"},
		{"F", "Administrative Costs", false, "OTHER_COST", "EXCLUSIVE"},
		{"G", "Gross Profit", true, "", ""},
		{"H", "Net Income", true, "", ""},
		{"I", "Tax Amount", true, "", ""},
		{"J", "Final Total", true, "", ""},
	}

	numFields := config.NumFieldsPerForm
	if numFields > len(fieldTemplates) {
		numFields = len(fieldTemplates)
	}

	for i := 0; i < numFields; i++ {
		field := fieldTemplates[i]
		fieldID := uuid.New()
		
		var sectionType, taxType, paymentResp *string
		var coaIDPtr *uuid.UUID
		
		if !field.isComputed {
			sectionType = &field.sectionType
			taxType = &field.taxType
			coaIDPtr = &coaID
			pr := gofakeit.RandomString([]string{"OWNER", "CLINIC"})
			paymentResp = &pr
		}
		
		_, err = tx.ExecContext(ctx, `
			INSERT INTO tbl_form_field (id, form_version_id, field_key, label, is_computed, is_formula, 
				section_type, payment_responsibility, tax_type, coa_id, sort_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, fieldID, versionID, field.key, field.label, field.isComputed, field.isComputed,
			sectionType, paymentResp, taxType, coaIDPtr, i+1, time.Now(), time.Now())
		
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("failed to insert form field: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, "", err
	}

	return formID, formName, nil
}

func createBasicCOAAdvanced(ctx context.Context, tx *sqlx.Tx, practitionerID uuid.UUID) (uuid.UUID, error) {
	// Create multiple COA entries
	coaEntries := []struct {
		code          int
		name          string
		accountTypeID int
		accountTaxID  int
	}{
		{4000, "Revenue", 4, 1},
		{5000, "Cost of Goods Sold", 5, 1},
		{6000, "Operating Expenses", 5, 1},
		{7000, "Other Income", 4, 1},
	}

	var firstCoaID uuid.UUID
	
	for i, entry := range coaEntries {
		coaID := uuid.New()
		if i == 0 {
			firstCoaID = coaID
		}
		
		_, err := tx.ExecContext(ctx, `
			INSERT INTO tbl_chart_of_accounts (id, practitioner_id, code, name, account_type_id, account_tax_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, coaID, practitionerID, entry.code, entry.name, entry.accountTypeID, entry.accountTaxID, time.Now(), time.Now())
		
		if err != nil {
			return uuid.Nil, err
		}
	}
	
	return firstCoaID, nil
}

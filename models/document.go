package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ApprovalLevel represents a single level in the approval hierarchy
type ApprovalLevel struct {
	RoleName   string              `json:"roleName" bson:"roleName"` // "SH", "BR", "DH", "GDH"
	Status     string              `json:"status" bson:"status"`     // "pending", "approved", "rejected"
	ApproverID *primitive.ObjectID `json:"approverId,omitempty" bson:"approverId,omitempty"`
	ApprovedAt *time.Time          `json:"approvedAt,omitempty" bson:"approvedAt,omitempty"`
	RejectedAt *time.Time          `json:"rejectedAt,omitempty" bson:"rejectedAt,omitempty"`
	Comments   string              `json:"comments,omitempty" bson:"comments,omitempty"` // Optional comments from approver
}

// Document merepresentasikan sebuah dokumen teks di database.
type Document struct {
	BaseModel            `json:",inline" bson:",inline"` // Menyematkan ID, CreatedOn, CreatedBy, dll.
	Title                string                          `json:"title" bson:"title"`     // Judul dokumen
	Content              string                          `json:"content" bson:"content"` // Konten dokumen dalam format JSON string dari Lexical
	OwnerID              primitive.ObjectID              `json:"ownerId" bson:"ownerId"` // ID pengguna yang memiliki dokumen
	Status               string                          `json:"status" bson:"status"`
	DocNo                string                          `json:"docNo" bson:"docNo"`
	Version              float64                         `json:"version" bson:"version"` // Back to float64 as the correct type
	Priority             string                          `json:"priority" bson:"priority"`
	CurrentApprovalLevel int                             `json:"currentApprovalLevel" bson:"currentApprovalLevel"` // 0 = draft/not submitted, 1 = SH, 2 = BR, 3 = DH/GDH
	Approvals            []ApprovalLevel                 `json:"approvals" bson:"approvals"`                       // Sequential approval chain
}

// InitializeApprovalWorkflow initializes the approval workflow for a new document
func (d *Document) InitializeApprovalWorkflow() {
	d.CurrentApprovalLevel = 1 // Start at level 1 (SH) so it appears in SH's pending approvals
	d.Approvals = []ApprovalLevel{
		{
			RoleName: "SH", // Section Head
			Status:   "pending",
		},
		{
			RoleName: "BR", // Business Requirement
			Status:   "pending",
		},
		{
			RoleName: "DH", // Department Head / Group Department Head
			Status:   "pending",
		},
	}
	d.Status = "Ready for Review" // Initial status when workflow starts
}

// GetCurrentApprovalLevel returns the current approval level details
func (d *Document) GetCurrentApprovalLevel() *ApprovalLevel {
	if d.CurrentApprovalLevel == 0 || len(d.Approvals) == 0 {
		return nil
	}

	index := d.CurrentApprovalLevel - 1
	if index >= 0 && index < len(d.Approvals) {
		return &d.Approvals[index]
	}

	return nil
}

// CanUserApprove checks if a user with given role can approve at current level
func (d *Document) CanUserApprove(userRole string) bool {
	currentLevel := d.GetCurrentApprovalLevel()
	if currentLevel == nil {
		return false
	}

	// Check role-based approval permissions
	switch currentLevel.RoleName {
	case "SH":
		// SH or DH can approve at SH level (FR-5.4.2.3: DH has equivalent access to SH)
		return userRole == "SH" || userRole == "DH"
	case "BR":
		// Only BR can approve at BR level
		return userRole == "BR"
	case "DH":
		// Both DH and GDH can approve at DH level (final approval)
		return userRole == "DH" || userRole == "GDH"
	default:
		return false
	}
}

// ApproveCurrentLevel approves the current approval level
func (d *Document) ApproveCurrentLevel(approverID primitive.ObjectID, comments string) error {
	currentLevel := d.GetCurrentApprovalLevel()
	if currentLevel == nil {
		return fmt.Errorf("no current approval level found")
	}

	now := time.Now()
	currentLevel.Status = "approved"
	currentLevel.ApproverID = &approverID
	currentLevel.ApprovedAt = &now
	currentLevel.Comments = comments

	// Move to next level or mark as final approved
	if d.CurrentApprovalLevel < len(d.Approvals) {
		d.CurrentApprovalLevel++

		// Update document status based on current level
		if d.CurrentApprovalLevel > len(d.Approvals) {
			d.Status = "Final Approved"
		} else {
			nextLevel := d.GetCurrentApprovalLevel()
			if nextLevel != nil {
				switch nextLevel.RoleName {
				case "BR":
					d.Status = "Menunggu persetujuan BR"
				case "DH":
					d.Status = "Menunggu persetujuan DH"
				}
			}
		}
	}

	return nil
}

// RejectCurrentLevel rejects the current approval level
func (d *Document) RejectCurrentLevel(approverID primitive.ObjectID, comments string) error {
	currentLevel := d.GetCurrentApprovalLevel()
	if currentLevel == nil {
		return fmt.Errorf("no current approval level found")
	}

	now := time.Now()
	currentLevel.Status = "rejected"
	currentLevel.ApproverID = &approverID
	currentLevel.RejectedAt = &now
	currentLevel.Comments = comments

	d.Status = "Rejected"

	return nil
}

// IsFullyApproved checks if document has been approved by all levels
func (d *Document) IsFullyApproved() bool {
	if len(d.Approvals) == 0 {
		return false
	}

	for _, approval := range d.Approvals {
		if approval.Status != "approved" {
			return false
		}
	}

	return true
}

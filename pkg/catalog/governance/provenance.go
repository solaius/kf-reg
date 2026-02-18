package governance

// ProvenanceExtractor extracts provenance information from a provider source.
// Providers (YAML, Git, HTTP, OCI) implement this to supply source metadata
// for version snapshots.
type ProvenanceExtractor interface {
	// ExtractProvenance returns provenance information for the given asset.
	// Returns nil if provenance cannot be determined (the version will still
	// be created without provenance).
	ExtractProvenance(plugin, kind, name string) *ProvenanceInfo
}

// StaticProvenanceExtractor returns fixed provenance info. Used for providers
// that can determine provenance at ingest time (e.g., Git with known repo URL
// and commit SHA).
type StaticProvenanceExtractor struct {
	SourceType string
	SourceURI  string
	SourceID   string
	RevisionID string
}

// ExtractProvenance returns the static provenance info regardless of asset identity.
func (e *StaticProvenanceExtractor) ExtractProvenance(plugin, kind, name string) *ProvenanceInfo {
	return &ProvenanceInfo{
		SourceType: e.SourceType,
		SourceURI:  e.SourceURI,
		SourceID:   e.SourceID,
		RevisionID: e.RevisionID,
	}
}

// ContentHashProvenanceExtractor generates provenance without a revision ID.
// Used by providers where the "revision" is determined externally (e.g., YAML
// files where the content hash serves as the revision marker).
type ContentHashProvenanceExtractor struct {
	SourceType string
	SourceURI  string // e.g., file path or URL
	SourceID   string // source ID from config
}

// ExtractProvenance returns provenance info with source metadata but no revision ID.
func (e *ContentHashProvenanceExtractor) ExtractProvenance(plugin, kind, name string) *ProvenanceInfo {
	return &ProvenanceInfo{
		SourceType: e.SourceType,
		SourceURI:  e.SourceURI,
		SourceID:   e.SourceID,
	}
}

// VerifyingProvenanceExtractor wraps another ProvenanceExtractor and adds
// integrity verification by computing a content hash. If the hash computation
// succeeds, the version is marked as verified; if it fails, the version is
// marked as unverified with the error in the details field.
type VerifyingProvenanceExtractor struct {
	Inner       ProvenanceExtractor
	Method      string // e.g., "sha256-content-hash"
	ComputeHash func(plugin, kind, name string) (string, error)
}

// ExtractProvenance delegates to the inner extractor and attaches integrity info.
func (e *VerifyingProvenanceExtractor) ExtractProvenance(plugin, kind, name string) *ProvenanceInfo {
	prov := e.Inner.ExtractProvenance(plugin, kind, name)
	if prov == nil {
		return nil
	}
	hash, err := e.ComputeHash(plugin, kind, name)
	if err != nil {
		prov.Integrity = &IntegrityInfo{
			Verified: false,
			Method:   e.Method,
			Details:  err.Error(),
		}
	} else {
		prov.Integrity = &IntegrityInfo{
			Verified: true,
			Method:   e.Method,
			Details:  hash,
		}
	}
	return prov
}

// applyProvenance populates provenance fields on a version record from extracted
// provenance info. If prov is nil, no fields are modified.
func applyProvenance(record *AssetVersionRecord, prov *ProvenanceInfo) {
	if prov == nil {
		return
	}
	record.ProvenanceSourceType = prov.SourceType
	record.ProvenanceSourceURI = prov.SourceURI
	record.ProvenanceSourceID = prov.SourceID
	record.ProvenanceRevisionID = prov.RevisionID
	if prov.Integrity != nil {
		record.ProvenanceVerified = prov.Integrity.Verified
		record.ProvenanceMethod = prov.Integrity.Method
		record.ProvenanceDetails = prov.Integrity.Details
	}
}

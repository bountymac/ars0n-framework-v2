import { useState, useRef, useCallback, useEffect } from 'react';
import { Modal, Table, Button, Spinner, Alert, Row, Col, Form, InputGroup } from 'react-bootstrap';
import { FaCheck, FaTimes } from 'react-icons/fa';

const AmassEnumConfigModal = ({ 
  show, 
  handleClose, 
  consolidatedCompanyDomains = [], 
  activeTarget,
  onSaveConfig
}) => {
  const [selectedDomains, setSelectedDomains] = useState(new Set());
  const [filters, setFilters] = useState({
    domain: ''
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [estimatedTime, setEstimatedTime] = useState(0);
  const [localDomains, setLocalDomains] = useState([]);
  const [loadingDomains, setLoadingDomains] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStartIndex, setDragStartIndex] = useState(null);
  const [dragMode, setDragMode] = useState('select');
  const tableRef = useRef(null);

  // Use consolidated domains from props, or fallback to locally fetched ones
  const domainsToUse = consolidatedCompanyDomains.length > 0 ? consolidatedCompanyDomains : localDomains;

  useEffect(() => {
    if (show) {
      loadSavedConfig();
      // If no domains provided via props, fetch them
      if (consolidatedCompanyDomains.length === 0) {
        fetchConsolidatedDomains();
      }
    }
  }, [show, activeTarget]);

  useEffect(() => {
    setEstimatedTime(selectedDomains.size);
  }, [selectedDomains]);

  const fetchConsolidatedDomains = async () => {
    if (!activeTarget?.id) return;

    setLoadingDomains(true);
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/consolidated-company-domains/${activeTarget.id}`
      );
      
      if (response.ok) {
        const data = await response.json();
        if (data.domains && Array.isArray(data.domains)) {
          setLocalDomains(data.domains);
        }
      }
    } catch (error) {
      console.error('Error fetching consolidated domains:', error);
      setError('Failed to load domains. Please try again.');
    } finally {
      setLoadingDomains(false);
    }
  };

  const loadSavedConfig = async () => {
    if (!activeTarget?.id) return;

    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-enum-config/${activeTarget.id}`
      );
      
      if (response.ok) {
        const config = await response.json();
        if (config.domains && Array.isArray(config.domains)) {
          setSelectedDomains(new Set(config.domains));
        }
      }
    } catch (error) {
      console.error('Error loading Amass Enum config:', error);
    }
  };

  const handleSaveConfig = async () => {
    if (!activeTarget?.id) {
      setError('No active target selected');
      return;
    }

    setSaving(true);
    setError('');

    try {
      const config = {
        domains: Array.from(selectedDomains),
        created_at: new Date().toISOString()
      };

      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-enum-config/${activeTarget.id}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(config),
        }
      );

      if (!response.ok) {
        throw new Error('Failed to save configuration');
      }

      if (onSaveConfig) {
        onSaveConfig(config);
      }

      handleClose();
    } catch (error) {
      console.error('Error saving Amass Enum config:', error);
      setError('Failed to save configuration. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({
      ...prev,
      [filterKey]: value
    }));
  };

  const clearFilters = () => {
    setFilters({
      domain: ''
    });
  };

  const handleDomainSelect = (domain, index) => {
    const newSelected = new Set(selectedDomains);
    if (newSelected.has(domain)) {
      newSelected.delete(domain);
    } else {
      newSelected.add(domain);
    }
    setSelectedDomains(newSelected);
  };

  const handleMouseDown = (domain, index, event) => {
    if (event.button !== 0) return;
    
    setIsDragging(true);
    setDragStartIndex(index);
    
    const newSelected = new Set(selectedDomains);
    const wasSelected = newSelected.has(domain);
    
    if (wasSelected) {
      newSelected.delete(domain);
      setDragMode('deselect');
    } else {
      newSelected.add(domain);
      setDragMode('select');
    }
    
    setSelectedDomains(newSelected);
    event.preventDefault();
  };

  const handleMouseEnter = useCallback((domain, index) => {
    if (!isDragging || dragStartIndex === null) return;
    
    const filteredDomains = getFilteredAndSortedDomains();
    const startIndex = Math.min(dragStartIndex, index);
    const endIndex = Math.max(dragStartIndex, index);
    
    const newSelected = new Set(selectedDomains);
    for (let i = startIndex; i <= endIndex; i++) {
      if (i < filteredDomains.length) {
        const domainAtIndex = typeof filteredDomains[i] === 'string' ? filteredDomains[i] : filteredDomains[i].domain;
        if (dragMode === 'select') {
          newSelected.add(domainAtIndex);
        } else {
          newSelected.delete(domainAtIndex);
        }
      }
    }
    setSelectedDomains(newSelected);
  }, [isDragging, dragStartIndex, selectedDomains, dragMode]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
    setDragStartIndex(null);
    setDragMode('select');
  }, []);

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mouseup', handleMouseUp);
      return () => {
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isDragging, handleMouseUp]);

  const selectAllFiltered = () => {
    const filteredDomains = getFilteredAndSortedDomains();
    const allDomains = filteredDomains.map(item => typeof item === 'string' ? item : item.domain);
    setSelectedDomains(new Set([...selectedDomains, ...allDomains]));
  };

  const deselectAllFiltered = () => {
    const filteredDomains = getFilteredAndSortedDomains();
    const filteredDomainsSet = new Set(filteredDomains.map(item => typeof item === 'string' ? item : item.domain));
    const newSelectedDomains = new Set([...selectedDomains].filter(domain => !filteredDomainsSet.has(domain)));
    setSelectedDomains(newSelectedDomains);
  };

  const handleSelectAll = () => {
    const filteredDomains = getFilteredAndSortedDomains();
    const allDomains = filteredDomains.map(item => typeof item === 'string' ? item : item.domain);
    setSelectedDomains(new Set(allDomains));
  };

  const handleDeselectAll = () => {
    setSelectedDomains(new Set());
  };

  const getFilteredAndSortedDomains = () => {
    let filteredDomains = domainsToUse.filter(item => {
      const domain = typeof item === 'string' ? item : item.domain;
      if (!domain) return false;
      
      if (filters.domain && !domain.toLowerCase().includes(filters.domain.toLowerCase())) {
        return false;
      }
      
      return true;
    });

    return filteredDomains;
  };

  const handleCloseModal = () => {
    setError('');
    setIsDragging(false);
    setDragStartIndex(null);
    setDragMode('select');
    handleClose();
  };

  const filteredDomains = getFilteredAndSortedDomains();

  return (
    <Modal 
      show={show} 
      onHide={handleCloseModal} 
      size="xl" 
      data-bs-theme="dark"
      className="modal-90w"
    >
      <Modal.Header closeButton>
        <Modal.Title className="text-danger">
          <i className="bi bi-cloud-arrow-down me-2" />
          Configure Amass Enum - Cloud Asset Discovery
        </Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {error && (
          <Alert variant="danger" dismissible onClose={() => setError('')}>
            {error}
          </Alert>
        )}

        <div className="mb-4">
          <Alert variant="warning">
            <div className="d-flex align-items-center">
              <i className="bi bi-exclamation-triangle-fill me-2" />
              <div>
                <strong>Performance Warning:</strong> Amass Enum can take up to 1 hour per domain to complete.
                Selected domains: <strong>{selectedDomains.size}</strong> 
                | Estimated total time: <strong>{estimatedTime === 1 ? '~1 hour' : `~${estimatedTime} hours`}</strong>
              </div>
            </div>
          </Alert>
        </div>

        {domainsToUse.length === 0 ? (
          <div className="text-center py-4">
            {loadingDomains ? (
              <>
                <div className="spinner-border text-danger mb-3" role="status">
                  <span className="visually-hidden">Loading...</span>
                </div>
                <h5 className="text-white-50">Loading Domains...</h5>
                <p className="text-white-50">
                  Fetching consolidated company domains...
                </p>
              </>
            ) : (
              <>
                <i className="bi bi-cloud text-white-50" style={{ fontSize: '3rem' }} />
                <h5 className="text-white-50 mt-3">No Domains Available</h5>
                <p className="text-white-50">
                  Run the company domain discovery tools and consolidate the results first.
                </p>
              </>
            )}
          </div>
        ) : (
          <>
            <div className="d-flex justify-content-between align-items-center mb-3">
              <h6 className="mb-0 text-white">
                Select domains for Amass Enum cloud asset discovery 
                <span className="text-light ms-2">({selectedDomains.size}/{domainsToUse.length})</span>
              </h6>
            </div>

            <Row className="mb-3">
              <Col className="d-flex align-items-end gap-2">
                <Form.Group className="flex-grow-1">
                  <Form.Label className="text-white-50 small">Filter by Domain</Form.Label>
                  <InputGroup>
                    <InputGroup.Text>
                      <i className="bi bi-search" />
                    </InputGroup.Text>
                    <Form.Control
                      type="text"
                      placeholder="Search domains..."
                      value={filters.domain}
                      onChange={(e) => handleFilterChange('domain', e.target.value)}
                      data-bs-theme="dark"
                    />
                  </InputGroup>
                </Form.Group>
                <Button variant="outline-secondary" size="sm" onClick={clearFilters}>
                  Clear Filter
                </Button>
              </Col>
            </Row>

            <div className="d-flex mb-3" style={{ gap: '8px' }}>
              <Button
                variant="danger"
                size="sm"
                onClick={handleDeselectAll}
                disabled={selectedDomains.size === 0}
                style={{ flex: 1 }}
              >
                <FaTimes className="me-1" />
                De-Select All
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={handleSelectAll}
                disabled={filteredDomains.length === 0}
                style={{ flex: 1 }}
              >
                <FaCheck className="me-1" />
                Select All Filtered
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={selectAllFiltered}
                disabled={filteredDomains.length === 0}
                style={{ flex: 1 }}
              >
                <FaCheck className="me-1" />
                Select All Visible
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={deselectAllFiltered}
                disabled={selectedDomains.size === 0}
                style={{ flex: 1 }}
              >
                <FaTimes className="me-1" />
                Deselect All Visible
              </Button>
            </div>

            <div className="mb-3">
              <small className="text-white-50">
                Showing {filteredDomains.length} of {domainsToUse.length} domains
              </small>
            </div>

            <div 
              style={{ 
                maxHeight: '400px', 
                overflowY: 'auto',
                border: '1px solid var(--bs-border-color)',
                borderRadius: '0.375rem'
              }}
              ref={tableRef}
            >
              <style>
                {`
                  .form-check-input:checked {
                    background-color: #dc3545 !important;
                    border-color: #dc3545 !important;
                  }
                  .form-check-input:focus {
                    border-color: #dc3545 !important;
                    box-shadow: 0 0 0 0.25rem rgba(220, 53, 69, 0.25) !important;
                  }
                `}
              </style>
              <Table hover variant="dark" size="sm" className="mb-0">
                <thead style={{ position: 'sticky', top: 0, zIndex: 10 }}>
                  <tr>
                    <th width="40" style={{ backgroundColor: 'var(--bs-dark)' }}>
                      <Form.Check
                        type="checkbox"
                        checked={filteredDomains.length > 0 && filteredDomains.every(item => {
                          const domain = typeof item === 'string' ? item : item.domain;
                          return selectedDomains.has(domain);
                        })}
                        onChange={(e) => {
                          if (e.target.checked) {
                            handleSelectAll();
                          } else {
                            handleDeselectAll();
                          }
                        }}
                      />
                    </th>
                    <th style={{ backgroundColor: 'var(--bs-dark)' }}>Domain</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredDomains.map((item, index) => {
                    const domain = typeof item === 'string' ? item : item.domain;
                    const isSelected = selectedDomains.has(domain);
                    
                    return (
                      <tr 
                        key={index}
                        style={{
                          backgroundColor: isSelected 
                            ? 'rgba(220, 53, 69, 0.25)' 
                            : 'transparent',
                          cursor: 'pointer',
                          userSelect: 'none',
                          transition: 'background-color 0.15s ease-in-out'
                        }}
                        onMouseDown={(e) => handleMouseDown(domain, index, e)}
                        onMouseEnter={() => handleMouseEnter(domain, index)}
                      >
                        <td>
                          <Form.Check
                            type="checkbox"
                            checked={isSelected}
                            onChange={() => handleDomainSelect(domain, index)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </td>
                        <td 
                          style={{ 
                            fontFamily: 'monospace',
                            fontSize: '0.875rem'
                          }}
                        >
                          {domain}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </Table>
            </div>

            {filteredDomains.length === 0 && (
              <div className="text-center py-4">
                <i className="bi bi-funnel text-white-50" style={{ fontSize: '2rem' }} />
                <h6 className="text-white-50 mt-2">No domains match the current filters</h6>
                <Button variant="outline-secondary" size="sm" onClick={clearFilters}>
                  Clear Filters
                </Button>
              </div>
            )}
          </>
        )}
      </Modal.Body>
      <Modal.Footer>
        <div className="d-flex justify-content-between align-items-center w-100">
          <div className="text-white-50 small">
            {selectedDomains.size > 0 && (
              <>
                <i className="bi bi-clock me-1" />
                Estimated scan time: {estimatedTime === 1 ? '~1 hour' : `~${estimatedTime} hours`}
              </>
            )}
          </div>
          <div>
            <Button variant="secondary" onClick={handleCloseModal} className="me-2">
              Cancel
            </Button>
            <Button 
              variant="danger" 
              onClick={handleSaveConfig}
              disabled={saving || selectedDomains.size === 0}
            >
              {saving ? (
                <>
                  <Spinner animation="border" size="sm" className="me-2" />
                  Saving...
                </>
              ) : (
                <>
                  <i className="bi bi-save me-2" />
                  Save Configuration ({selectedDomains.size} domains)
                </>
              )}
            </Button>
          </div>
        </div>
      </Modal.Footer>
    </Modal>
  );
};

export default AmassEnumConfigModal; 
import { useState, useEffect } from 'react';
import { Modal, Table, Button, Badge, Spinner, Alert, Row, Col, Form, InputGroup } from 'react-bootstrap';

const AmassEnumConfigModal = ({ 
  show, 
  handleClose, 
  consolidatedCompanyDomains = [], 
  activeTarget,
  onSaveConfig
}) => {
  const [selectedDomains, setSelectedDomains] = useState(new Set());
  const [filters, setFilters] = useState({
    domain: '',
    status: ''
  });
  const [sortColumn, setSortColumn] = useState('');
  const [sortDirection, setSortDirection] = useState('asc');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [estimatedTime, setEstimatedTime] = useState(0);
  const [localDomains, setLocalDomains] = useState([]);
  const [loadingDomains, setLoadingDomains] = useState(false);

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
      domain: '',
      status: ''
    });
  };

  const toggleDomainSelection = (domain) => {
    const newSelectedDomains = new Set(selectedDomains);
    if (newSelectedDomains.has(domain)) {
      newSelectedDomains.delete(domain);
    } else {
      newSelectedDomains.add(domain);
    }
    setSelectedDomains(newSelectedDomains);
  };

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

  const getFilteredAndSortedDomains = () => {
    let filteredDomains = domainsToUse.filter(item => {
      const domain = typeof item === 'string' ? item : item.domain;
      if (!domain) return false;
      
      if (filters.domain && !domain.toLowerCase().includes(filters.domain.toLowerCase())) {
        return false;
      }
      
      if (filters.status) {
        const isSelected = selectedDomains.has(domain);
        if (filters.status === 'selected' && !isSelected) return false;
        if (filters.status === 'unselected' && isSelected) return false;
      }
      
      return true;
    });

    if (!sortColumn) return filteredDomains;

    return filteredDomains.sort((a, b) => {
      const domainA = typeof a === 'string' ? a : a.domain;
      const domainB = typeof b === 'string' ? b : b.domain;

      let valueA, valueB;

      switch (sortColumn) {
        case 'domain':
          valueA = domainA || '';
          valueB = domainB || '';
          break;
        case 'status':
          valueA = selectedDomains.has(domainA) ? 1 : 0;
          valueB = selectedDomains.has(domainB) ? 1 : 0;
          break;
        default:
          return 0;
      }

      if (valueA < valueB) return sortDirection === 'asc' ? -1 : 1;
      if (valueA > valueB) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  };

  const handleSort = (column) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const renderSortIcon = (column) => {
    if (sortColumn !== column) {
      return <i className="bi bi-arrow-down-up text-muted ms-1"></i>;
    }
    return sortDirection === 'asc' ? 
      <i className="bi bi-arrow-up text-primary ms-1" /> : 
      <i className="bi bi-arrow-down text-primary ms-1" />;
  };

  const handleCloseModal = () => {
    setError('');
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
          <div>
            <div className="mb-3">
              <h6 className="text-white">
                Select domains for Amass Enum cloud asset discovery 
                ({selectedDomains.size} of {domainsToUse.length} selected)
              </h6>
              <p className="text-white-50 small">
                Amass Enum will perform comprehensive DNS enumeration and cloud asset discovery 
                on the selected domains using active techniques, brute-forcing, and multiple data sources.
              </p>
            </div>

            <Row className="mb-3">
              <Col md={4}>
                <Form.Group>
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
              </Col>
              <Col md={3}>
                <Form.Group>
                  <Form.Label className="text-white-50 small">Selection Status</Form.Label>
                  <Form.Select
                    value={filters.status}
                    onChange={(e) => handleFilterChange('status', e.target.value)}
                    data-bs-theme="dark"
                  >
                    <option value="">All Domains</option>
                    <option value="selected">Selected Only</option>
                    <option value="unselected">Unselected Only</option>
                  </Form.Select>
                </Form.Group>
              </Col>
              <Col md={5} className="d-flex align-items-end gap-2">
                <Button variant="outline-success" size="sm" onClick={selectAllFiltered}>
                  Select All Filtered
                </Button>
                <Button variant="outline-warning" size="sm" onClick={deselectAllFiltered}>
                  Deselect All Filtered
                </Button>
                <Button variant="outline-secondary" size="sm" onClick={clearFilters}>
                  Clear Filters
                </Button>
              </Col>
            </Row>

            <div className="mb-3">
              <small className="text-white-50">
                Showing {filteredDomains.length} of {domainsToUse.length} domains
              </small>
            </div>

            <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
              <Table striped bordered hover variant="dark" className="mb-0">
                <thead className="sticky-top">
                  <tr>
                    <th style={{ width: '50px' }}>
                      <Form.Check
                        type="checkbox"
                        checked={filteredDomains.length > 0 && filteredDomains.every(item => {
                          const domain = typeof item === 'string' ? item : item.domain;
                          return selectedDomains.has(domain);
                        })}
                        onChange={(e) => {
                          if (e.target.checked) {
                            selectAllFiltered();
                          } else {
                            deselectAllFiltered();
                          }
                        }}
                      />
                    </th>
                    <th 
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleSort('domain')}
                    >
                      Domain
                      {renderSortIcon('domain')}
                    </th>
                    <th 
                      style={{ cursor: 'pointer', width: '150px' }}
                      onClick={() => handleSort('status')}
                    >
                      Status
                      {renderSortIcon('status')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {filteredDomains.map((item, index) => {
                    const domain = typeof item === 'string' ? item : item.domain;
                    const isSelected = selectedDomains.has(domain);
                    
                    return (
                      <tr key={index}>
                        <td>
                          <Form.Check
                            type="checkbox"
                            checked={isSelected}
                            onChange={() => toggleDomainSelection(domain)}
                          />
                        </td>
                        <td>
                          <code className={isSelected ? 'text-success' : 'text-white'}>
                            {domain}
                          </code>
                        </td>
                        <td>
                          {isSelected ? (
                            <Badge bg="success">
                              <i className="bi bi-check-circle me-1" />
                              Selected
                            </Badge>
                          ) : (
                            <Badge bg="secondary">
                              <i className="bi bi-circle me-1" />
                              Not Selected
                            </Badge>
                          )}
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
          </div>
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
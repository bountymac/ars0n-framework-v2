import { useState, useEffect, useMemo } from 'react';
import { Modal, Button, Badge, ListGroup, Row, Col, Card, Alert, Table } from 'react-bootstrap';
import { copyToClipboard } from '../utils/miscUtils';

export const NucleiResultsModal = ({ 
  show, 
  handleClose, 
  scan,
  setShowToast 
}) => {
  const [selectedFinding, setSelectedFinding] = useState(null);
  const [findings, setFindings] = useState([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [severityFilter, setSeverityFilter] = useState('all');
  const [categoryFilter, setCategoryFilter] = useState('all');

  const formatResults = (results) => {
    console.log('[NucleiResultsModal] Formatting results:', results);
    
    if (!results?.result) {
      console.log('[NucleiResultsModal] No result data found');
      return [];
    }
    
    try {
      let findings = [];
      if (typeof results.result === 'string') {
        console.log('[NucleiResultsModal] Parsing string result');
        findings = JSON.parse(results.result);
      } else if (Array.isArray(results.result)) {
        console.log('[NucleiResultsModal] Result is already an array');
        findings = results.result;
      }
      
      console.log('[NucleiResultsModal] Parsed findings:', findings);
      console.log('[NucleiResultsModal] Findings count:', Array.isArray(findings) ? findings.length : 0);
      
      if (Array.isArray(findings) && findings.length > 0) {
        console.log('[NucleiResultsModal] Sample finding:', findings[0]);
        console.log('[NucleiResultsModal] Sample finding severity:', findings[0].info?.severity);
      }
      
      return Array.isArray(findings) ? findings : [];
    } catch (error) {
      console.error('Error parsing Nuclei results:', error);
      return [];
    }
  };

  useEffect(() => {
    if (scan) {
      const formattedResults = formatResults(scan);
      setFindings(formattedResults);
      if (formattedResults.length > 0) {
        setSelectedFinding(formattedResults[0]);
      } else {
        setSelectedFinding(null);
      }
    } else {
      setFindings([]);
      setSelectedFinding(null);
    }
  }, [scan]);

  const handleCopy = async () => {
    if (scan?.result) {
      try {
        const exportText = findings.map(f => 
          `[${f.info?.severity?.toUpperCase() || 'INFO'}] ${f.info?.name || 'Unknown'} - ${f.host || f.matched}\n` +
          `Template: ${f.template_id || 'N/A'}\n` +
          `Matcher: ${f.matcher_name || 'N/A'}\n` +
          `${f.info?.description ? `Description: ${f.info.description}\n` : ''}` +
          `---\n`
        ).join('\n');
        
        const success = await copyToClipboard(exportText);
        if (success && setShowToast) {
          setShowToast(true);
          setTimeout(() => setShowToast(false), 3000);
        }
      } catch (error) {
        console.error('Error copying results:', error);
      }
    }
  };

  const handleCopyFinding = async (finding) => {
    try {
      const exportText = 
        `[${finding.info?.severity?.toUpperCase() || 'INFO'}] ${finding.info?.name || 'Unknown'}\n` +
        `Template: ${finding.template_id || 'N/A'}\n` +
        `Target: ${finding.host || finding.matched || 'N/A'}\n` +
        `Matcher: ${finding.matcher_name || 'N/A'}\n` +
        `${finding.info?.description ? `Description: ${finding.info.description}\n` : ''}` +
        `${finding.info?.reference ? `References: ${finding.info.reference.join(', ')}\n` : ''}` +
        `${finding.info?.tags ? `Tags: ${finding.info.tags.join(', ')}\n` : ''}`;
      
      const success = await copyToClipboard(exportText);
      if (success && setShowToast) {
        setShowToast(true);
        setTimeout(() => setShowToast(false), 3000);
      }
    } catch (error) {
      console.error('Error copying finding:', error);
    }
  };

  const getSeverityBadge = (severity) => {
    const severityMap = {
      'critical': 'danger',
      'high': 'danger',
      'medium': 'warning', 
      'low': 'info',
      'info': 'secondary'
    };
    return severityMap[severity?.toLowerCase()] || 'secondary';
  };

  const getSeverityIcon = (severity) => {
    const iconMap = {
      'critical': 'exclamation-triangle-fill',
      'high': 'exclamation-triangle',
      'medium': 'exclamation-circle',
      'low': 'info-circle',
      'info': 'info-circle-fill'
    };
    return iconMap[severity?.toLowerCase()] || 'info-circle';
  };

  const groupBySeverity = (findings) => {
    const grouped = findings.reduce((acc, finding) => {
      let severity = finding.info?.severity?.toLowerCase() || 'info';
      
      if (severity === 'unknown') {
        severity = 'info';
      }
      
      if (!acc[severity]) acc[severity] = [];
      acc[severity].push(finding);
      return acc;
    }, {});
    
    const severityOrder = ['critical', 'high', 'medium', 'low', 'info'];
    const sortedGrouped = {};
    severityOrder.forEach(severity => {
      if (grouped[severity]) {
        sortedGrouped[severity] = grouped[severity];
      }
    });
    
    return sortedGrouped;
  };

  const getAvailableCategories = useMemo(() => {
    const categories = new Set();
    findings.forEach(finding => {
      if (finding.info?.tags && finding.info.tags.length > 0) {
        finding.info.tags.forEach(tag => categories.add(tag));
      }
    });
    return Array.from(categories).sort();
  }, [findings]);

  const filteredFindings = useMemo(() => {
    let filtered = findings;

    if (searchTerm) {
      const isNegativeSearch = searchTerm.startsWith('-');
      const searchValue = isNegativeSearch ? searchTerm.substring(1) : searchTerm;
      
      filtered = filtered.filter(finding => {
        const searchableText = [
          finding.info?.name || '',
          finding.host || '',
          finding.matched || '',
          finding.template_id || '',
          finding.info?.description || '',
          finding.info?.tags?.join(' ') || ''
        ].join(' ').toLowerCase();
        
        const matches = searchableText.includes(searchValue.toLowerCase());
        return isNegativeSearch ? !matches : matches;
      });
    }

    if (severityFilter !== 'all') {
      filtered = filtered.filter(finding => {
        const severity = finding.info?.severity?.toLowerCase() || 'info';
        return severity === severityFilter || (severity === 'unknown' && severityFilter === 'info');
      });
    }

    if (categoryFilter !== 'all') {
      filtered = filtered.filter(finding => {
        return finding.info?.tags?.includes(categoryFilter);
      });
    }

    return filtered;
  }, [findings, searchTerm, severityFilter, categoryFilter]);

  const filteredGroupedFindings = useMemo(() => {
    return groupBySeverity(filteredFindings);
  }, [filteredFindings]);

  const allFindings = useMemo(() => {
    const findings = [];
    Object.entries(filteredGroupedFindings).forEach(([severity, severityFindings]) => {
      severityFindings.forEach(finding => {
        findings.push({ ...finding, severity });
      });
    });
    return findings;
  }, [filteredGroupedFindings]);

  useEffect(() => {
    if (show && allFindings.length > 0 && !selectedFinding) {
      setSelectedFinding(allFindings[0]);
      setSelectedIndex(0);
    }
  }, [show, allFindings, selectedFinding]);

  useEffect(() => {
    const handleGlobalKeyDown = (event) => {
      if (!show) return;
      
      if (allFindings.length === 0) return;

      switch (event.key) {
        case 'ArrowDown':
          event.preventDefault();
          const nextIndex = Math.min(selectedIndex + 1, allFindings.length - 1);
          if (nextIndex !== selectedIndex) {
            setSelectedIndex(nextIndex);
            setSelectedFinding(allFindings[nextIndex]);
          }
          break;
        case 'ArrowUp':
          event.preventDefault();
          const prevIndex = Math.max(selectedIndex - 1, 0);
          if (prevIndex !== selectedIndex) {
            setSelectedIndex(prevIndex);
            setSelectedFinding(allFindings[prevIndex]);
          }
          break;
        case 'Home':
          event.preventDefault();
          if (selectedIndex !== 0) {
            setSelectedIndex(0);
            setSelectedFinding(allFindings[0]);
          }
          break;
        case 'End':
          event.preventDefault();
          const lastIndex = allFindings.length - 1;
          if (selectedIndex !== lastIndex) {
            setSelectedIndex(lastIndex);
            setSelectedFinding(allFindings[lastIndex]);
          }
          break;
      }
    };

    if (show) {
      document.addEventListener('keydown', handleGlobalKeyDown);
    }

    return () => {
      document.removeEventListener('keydown', handleGlobalKeyDown);
    };
  }, [show, allFindings, selectedIndex]);

  const renderFindingsList = () => {
    if (filteredFindings.length === 0) {
      return (
        <div className="text-center text-muted p-4">
          <i className="bi bi-search fs-1 mb-3 d-block"></i>
          <p>{findings.length === 0 ? 'No security findings detected in this scan.' : 'No findings match the current filters.'}</p>
        </div>
      );
    }

    return (
      <div style={{ height: '75vh', overflowY: 'auto' }}>
        {Object.entries(filteredGroupedFindings).map(([severity, severityFindings]) => (
          <div key={severity} className="mb-3">
            <div className="d-flex align-items-center mb-2">
              <Badge bg={getSeverityBadge(severity)} className="me-2">
                {severity.toUpperCase()}
              </Badge>
              <small className="text-muted">
                {severityFindings.length} finding{severityFindings.length !== 1 ? 's' : ''}
              </small>
            </div>
            
            <ListGroup variant="flush">
              {severityFindings.map((finding, index) => {
                const findingIndex = allFindings.findIndex(f => f === finding);
                const isSelected = selectedFinding === finding;
                
                return (
                  <ListGroup.Item
                    key={`${severity}-${index}`}
                    action
                    active={isSelected}
                    onClick={() => {
                      setSelectedFinding(finding);
                      setSelectedIndex(findingIndex);
                    }}
                    className="py-2 border-0 mb-1"
                    style={{ 
                      backgroundColor: isSelected ? 'rgba(13, 110, 253, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                      borderRadius: '4px',
                      border: isSelected ? '2px solid #0d6efd' : '2px solid transparent'
                    }}
                  >
                  <div className="d-flex align-items-start">
                    <i className={`bi bi-${getSeverityIcon(severity)} text-${getSeverityBadge(severity) === 'danger' ? 'danger' : getSeverityBadge(severity) === 'warning' ? 'warning' : 'info'} me-2 mt-1`}></i>
                    <div className="flex-grow-1">
                      <div className="fw-bold text-truncate" style={{ maxWidth: '200px' }}>
                        {finding.info?.name || finding.template_id || 'Unknown'}
                      </div>
                      <div className="text-muted small text-truncate" style={{ maxWidth: '200px' }}>
                        {finding.host || finding.matched || 'Unknown target'}
                      </div>
                      <div className="text-muted small">
                        {finding.template_id}
                      </div>
                    </div>
                  </div>
                </ListGroup.Item>
                );
              })}
            </ListGroup>
          </div>
        ))}
      </div>
    );
  };

  const renderFindingDetails = () => {
    if (!selectedFinding) {
      return (
        <div className="text-center text-muted p-4">
          <i className="bi bi-arrow-left fs-1 mb-3 d-block"></i>
          <p>Select a finding from the left to view details</p>
        </div>
      );
    }

    const finding = selectedFinding;
    const severity = finding.info?.severity?.toLowerCase() || 'info';

    return (
      <div style={{ height: '75vh', overflowY: 'auto' }}>
        <Card className="bg-dark border-secondary">
          <Card.Header className="d-flex justify-content-between align-items-center">
            <div className="d-flex align-items-center">
              <Badge bg={getSeverityBadge(severity)} className="me-2">
                {severity.toUpperCase()}
              </Badge>
              <span className="fw-bold">{finding.info?.name || finding.template_id || 'Unknown'}</span>
            </div>
            <Button 
              variant="outline-light" 
              size="sm" 
              onClick={() => handleCopyFinding(finding)}
              title="Copy finding details"
            >
              <i className="bi bi-clipboard"></i>
            </Button>
          </Card.Header>
          
          <Card.Body>
            <Row>
              <Col md={6}>
                <div className="mb-3">
                  <h6 className="text-light mb-2">
                    <i className="bi bi-bullseye me-2"></i>Target
                  </h6>
                  <div className="bg-secondary rounded p-2">
                    <div className="text-light">{finding.host || finding.matched || 'Unknown'}</div>
                    {finding.ip && finding.ip !== finding.host && (
                      <div className="text-muted small">IP: {finding.ip}</div>
                    )}
                    {finding.port && (
                      <div className="text-muted small">Port: {finding.port}</div>
                    )}
                  </div>
                </div>
              </Col>
              
              <Col md={6}>
                <div className="mb-3">
                  <h6 className="text-light mb-2">
                    <i className="bi bi-file-code me-2"></i>Template
                  </h6>
                  <div className="bg-secondary rounded p-2">
                    <div className="text-light">{finding.template_id || 'Unknown'}</div>
                    {finding.matcher_name && (
                      <div className="text-muted small">Matcher: {finding.matcher_name}</div>
                    )}
                  </div>
                </div>
              </Col>
            </Row>

            {finding.info?.description && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-info-circle me-2"></i>Description
                </h6>
                <Alert variant="info" className="mb-0">
                  {finding.info.description}
                </Alert>
              </div>
            )}

            {finding.info?.reference && finding.info.reference.length > 0 && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-link-45deg me-2"></i>References
                </h6>
                <div className="bg-secondary rounded p-2">
                  {finding.info.reference.map((ref, index) => (
                    <div key={index} className="mb-1">
                      <a 
                        href={ref} 
                        target="_blank" 
                        rel="noopener noreferrer" 
                        className="text-info text-decoration-none"
                      >
                        <i className="bi bi-link-45deg me-1"></i>
                        {ref}
                      </a>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {finding.info?.tags && finding.info.tags.length > 0 && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-tags me-2"></i>Tags
                </h6>
                <div>
                  {finding.info.tags.map((tag, index) => (
                    <Badge key={index} bg="secondary" className="me-1 mb-1">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {finding.info?.classification && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-diagram-3 me-2"></i>Classification
                </h6>
                <div className="bg-secondary rounded p-2">
                  {Object.entries(finding.info.classification).map(([key, value]) => (
                    <div key={key} className="mb-1">
                      <span className="text-muted">{key.toUpperCase()}:</span>
                      <span className="text-light ms-2">{Array.isArray(value) ? value.join(', ') : value}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {finding.extracted_results && finding.extracted_results.length > 0 && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-search me-2"></i>Extracted Results
                </h6>
                <div className="bg-secondary rounded p-2">
                  {finding.extracted_results.map((result, index) => (
                    <div key={index} className="mb-1">
                      <code className="text-warning">{result}</code>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {finding.curl_command && (
              <div className="mb-3">
                <h6 className="text-light mb-2">
                  <i className="bi bi-terminal me-2"></i>Curl Command
                </h6>
                <div className="bg-dark rounded p-2">
                  <code className="text-success small">{finding.curl_command}</code>
                </div>
              </div>
            )}
          </Card.Body>
        </Card>
      </div>
    );
  };

  return (
    <Modal 
      data-bs-theme="dark" 
      show={show} 
      onHide={handleClose} 
      size="xl"
      className="nuclei-results-modal"
      dialogClassName="modal-fullscreen"
      style={{ margin: 0 }}
      tabIndex={0}
    >
      <Modal.Header closeButton>
        <Modal.Title className='text-danger'>
          <i className="bi bi-shield-exclamation me-2"></i>
          Nuclei Scan Results - {findings.length} Finding{findings.length !== 1 ? 's' : ''}
        </Modal.Title>
      </Modal.Header>
      
      <Modal.Body className="p-0">
        <div className="bg-dark border-bottom px-3 py-3">
          <Row className="g-2">
            <Col md={4}>
              <div className="input-group input-group-sm">
                <span className="input-group-text bg-secondary border-secondary">
                  <i className="bi bi-search"></i>
                </span>
                <input
                  type="text"
                  className="form-control bg-dark text-light border-secondary"
                  placeholder="Search findings (use -term for negative search)"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
                {searchTerm && (
                  <button
                    className="btn btn-outline-secondary btn-sm"
                    onClick={() => setSearchTerm('')}
                  >
                    <i className="bi bi-x"></i>
                  </button>
                )}
              </div>
            </Col>
            <Col md={3}>
              <select
                className="form-select form-select-sm bg-dark text-light border-secondary"
                value={severityFilter}
                onChange={(e) => setSeverityFilter(e.target.value)}
              >
                <option value="all">All Severities</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
                <option value="info">Info</option>
              </select>
            </Col>
            <Col md={3}>
              <select
                className="form-select form-select-sm bg-dark text-light border-secondary"
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                <option value="all">All Categories</option>
                {getAvailableCategories.map(category => (
                  <option key={category} value={category}>{category}</option>
                ))}
              </select>
            </Col>
            <Col md={2}>
              <div className="d-flex justify-content-between align-items-center">
                <small className="text-muted">
                  {filteredFindings.length} of {findings.length}
                </small>
                <Button 
                  variant="outline-secondary" 
                  size="sm" 
                  onClick={() => {
                    setSearchTerm('');
                    setSeverityFilter('all');
                    setCategoryFilter('all');
                  }}
                  disabled={!searchTerm && severityFilter === 'all' && categoryFilter === 'all'}
                >
                  <i className="bi bi-arrow-clockwise"></i>
                </Button>
              </div>
            </Col>
          </Row>
        </div>
        
        <div className="d-flex align-items-center justify-content-between bg-dark border-bottom px-3 py-2">
          <div>
            <small className="text-muted">
              Scan ID: {scan?.id} | 
              Executed: {scan?.created_at ? new Date(scan.created_at).toLocaleString() : 'Unknown'}
            </small>
          </div>
          <Button variant="outline-success" size="sm" onClick={handleCopy} disabled={findings.length === 0}>
            <i className="bi bi-clipboard me-1"></i>
            Copy All Results
          </Button>
        </div>
        
        <Row className="g-0">
          <Col md={4} className="border-end">
            <div className="p-3">
              <div className="d-flex justify-content-between align-items-center mb-3">
                <h6 className="text-light mb-0">
                  <i className="bi bi-list-ul me-2"></i>Findings
                </h6>
                <small className="text-muted">
                  <i className="bi bi-keyboard me-1"></i>
                  Use ↑↓ arrows to navigate, Home/End for first/last
                </small>
              </div>
              {renderFindingsList()}
            </div>
          </Col>
          
          <Col md={8}>
            <div className="p-3">
              <h6 className="text-light mb-3">
                <i className="bi bi-info-circle me-2"></i>Details
              </h6>
              {renderFindingDetails()}
            </div>
          </Col>
        </Row>
      </Modal.Body>
      
      <Modal.Footer>
        <Button variant="secondary" onClick={handleClose}>
          Close
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export const NucleiHistoryModal = ({ 
  show, 
  handleClose, 
  scans 
}) => {
  const getFindingsCount = (scan) => {
    if (!scan?.result) return 0;
    try {
      if (typeof scan.result === 'string') {
        const parsed = JSON.parse(scan.result);
        return Array.isArray(parsed) ? parsed.length : 0;
      }
      return Array.isArray(scan.result) ? scan.result.length : 0;
    } catch (error) {
      return 0;
    }
  };

  const getErrorDisplay = (error) => {
    if (!error) return null;
    if (error.includes('timeout')) {
      return (
        <span className="text-warning" title="Scan timed out">
          <i className="bi bi-clock-fill me-1"></i>
          Timeout
        </span>
      );
    }
    return (
      <span className="text-danger" title={error}>
        <i className="bi bi-exclamation-triangle-fill me-1"></i>
        Error
      </span>
    );
  };

  const getStatusBadge = (status) => {
    const statusMap = {
      'success': 'success',
      'running': 'primary',
      'pending': 'warning',
      'failed': 'danger',
      'timeout': 'warning'
    };
    return statusMap[status] || 'secondary';
  };

  return (
    <Modal 
      data-bs-theme="dark" 
      show={show} 
      onHide={handleClose} 
      size="xl"
    >
      <Modal.Header closeButton>
        <Modal.Title className='text-danger'>
          <i className="bi bi-clock-history me-2"></i>
          Nuclei Scan History
        </Modal.Title>
      </Modal.Header>
      <Modal.Body style={{ maxHeight: '70vh', overflowY: 'auto' }}>
        {scans && scans.length > 0 ? (
          <Table striped bordered hover variant="dark">
            <thead>
              <tr>
                <th>Scan ID</th>
                <th>Status</th>
                <th>Findings</th>
                <th>Targets</th>
                <th>Templates</th>
                <th>Started</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              {scans.map((scan) => (
                <tr key={scan.id}>
                  <td>
                    <code className="text-info">{scan.id}</code>
                  </td>
                  <td>
                    <Badge bg={getStatusBadge(scan.status)}>
                      {scan.status}
                    </Badge>
                    {scan.error && (
                      <div className="mt-1">
                        {getErrorDisplay(scan.error)}
                      </div>
                    )}
                  </td>
                  <td>
                    <Badge bg={getFindingsCount(scan) > 0 ? 'danger' : 'success'}>
                      {getFindingsCount(scan)}
                    </Badge>
                  </td>
                  <td>
                    <Badge bg="info">
                      {scan.targets?.length || 0}
                    </Badge>
                  </td>
                  <td>
                    <Badge bg="secondary">
                      {scan.templates?.length || 0}
                    </Badge>
                  </td>
                  <td>
                    <small>
                      {scan.created_at ? new Date(scan.created_at).toLocaleString() : 'Unknown'}
                    </small>
                  </td>
                  <td>
                    <small>
                      {scan.execution_time ? `${scan.execution_time}s` : 'N/A'}
                    </small>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        ) : (
          <div className="text-center text-muted p-4">
            <i className="bi bi-clock-history fs-1 mb-3 d-block"></i>
            <p>No Nuclei scans found for this target.</p>
          </div>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={handleClose}>
          Close
        </Button>
      </Modal.Footer>
    </Modal>
  );
}; 
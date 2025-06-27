import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Modal, Button, Table, Form, Alert, InputGroup, Row, Col, Spinner } from 'react-bootstrap';
import { FaCheck, FaTimes } from 'react-icons/fa';

const DNSxConfigModal = ({ 
  show, 
  handleClose, 
  scopeTargets = [],
  consolidatedCompanyDomains = [],
  activeTarget,
  onSaveConfig
}) => {
  const [selectedWildcardTargets, setSelectedWildcardTargets] = useState(new Set());
  const [filters, setFilters] = useState({
    domain: ''
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [estimatedTime, setEstimatedTime] = useState(0);
  const [wildcardTargetsWithCounts, setWildcardTargetsWithCounts] = useState([]);
  const [loadingCounts, setLoadingCounts] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStartIndex, setDragStartIndex] = useState(null);
  const [dragMode, setDragMode] = useState('select');
  const tableRef = useRef(null);

  useEffect(() => {
    setEstimatedTime(selectedWildcardTargets.size * 1); // Estimate 1 hour per wildcard target
  }, [selectedWildcardTargets]);

  useEffect(() => {
    if (show) {
      loadSavedConfig();
      fetchWildcardTargetsWithCounts();
    }
  }, [show, activeTarget]);

  const getWildcardTargetsMatchingRootDomains = () => {
    return scopeTargets.filter(target => {
      if (target.type !== 'Wildcard' || !target.scope_target) return false;
      
      // Remove *. prefix if present to get the base domain
      const baseDomain = target.scope_target.startsWith('*.') 
        ? target.scope_target.substring(2) 
        : target.scope_target;
      
      // Check if this domain exists in consolidated company domains
      return consolidatedCompanyDomains.some(item => {
        const domain = typeof item === 'string' ? item : item.domain;
        return domain && domain.toLowerCase() === baseDomain.toLowerCase();
      });
    });
  };

  const fetchWildcardTargetsWithCounts = async () => {
    const wildcardTargets = getWildcardTargetsMatchingRootDomains();
    
    if (wildcardTargets.length === 0) {
      setWildcardTargetsWithCounts([]);
      return;
    }

    setLoadingCounts(true);
    try {
      const targetsWithCounts = await Promise.all(
        wildcardTargets.map(async (target) => {
          try {
            const response = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scope-target/${target.id}/live-web-servers-count`
            );
            
            if (response.ok) {
              const data = await response.json();
              return {
                ...target,
                liveWebServersCount: data.count || 0
              };
            } else {
              return {
                ...target,
                liveWebServersCount: 0
              };
            }
          } catch (error) {
            console.error(`Error fetching live web servers count for ${target.scope_target}:`, error);
            return {
              ...target,
              liveWebServersCount: 0
            };
          }
        })
      );
      
      setWildcardTargetsWithCounts(targetsWithCounts);
    } catch (error) {
      console.error('Error fetching wildcard targets with counts:', error);
      setError('Failed to load wildcard targets. Please try again.');
    } finally {
      setLoadingCounts(false);
    }
  };

  const loadSavedConfig = async () => {
    if (!activeTarget?.id) return;

    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/dnsx-config/${activeTarget.id}`
      );
      
      if (response.ok) {
        const config = await response.json();
        if (config.wildcard_targets && Array.isArray(config.wildcard_targets)) {
          setSelectedWildcardTargets(new Set(config.wildcard_targets));
        }
      }
    } catch (error) {
      console.error('Error loading DNSx config:', error);
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
        wildcard_targets: Array.from(selectedWildcardTargets),
        created_at: new Date().toISOString()
      };

      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/dnsx-config/${activeTarget.id}`,
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
      console.error('Error saving DNSx config:', error);
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

  const handleWildcardTargetSelect = (targetId, index) => {
    const newSelected = new Set(selectedWildcardTargets);
    if (newSelected.has(targetId)) {
      newSelected.delete(targetId);
    } else {
      newSelected.add(targetId);
    }
    setSelectedWildcardTargets(newSelected);
  };

  const handleMouseDown = (targetId, index, event) => {
    if (event.button !== 0) return;
    
    setIsDragging(true);
    setDragStartIndex(index);
    
    const newSelected = new Set(selectedWildcardTargets);
    const wasSelected = newSelected.has(targetId);
    
    if (wasSelected) {
      newSelected.delete(targetId);
      setDragMode('deselect');
    } else {
      newSelected.add(targetId);
      setDragMode('select');
    }
    
    setSelectedWildcardTargets(newSelected);
    event.preventDefault();
  };

  const handleMouseEnter = useCallback((targetId, index) => {
    if (!isDragging || dragStartIndex === null) return;
    
    const filteredTargets = getFilteredAndSortedWildcardTargets();
    const startIndex = Math.min(dragStartIndex, index);
    const endIndex = Math.max(dragStartIndex, index);
    
    const newSelected = new Set(selectedWildcardTargets);
    for (let i = startIndex; i <= endIndex; i++) {
      if (i < filteredTargets.length) {
        const targetAtIndex = filteredTargets[i].id;
        if (dragMode === 'select') {
          newSelected.add(targetAtIndex);
        } else {
          newSelected.delete(targetAtIndex);
        }
      }
    }
    setSelectedWildcardTargets(newSelected);
  }, [isDragging, dragStartIndex, selectedWildcardTargets, dragMode]);

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
    const filteredTargets = getFilteredAndSortedWildcardTargets();
    const allTargetIds = filteredTargets.map(target => target.id);
    setSelectedWildcardTargets(new Set([...selectedWildcardTargets, ...allTargetIds]));
  };

  const deselectAllFiltered = () => {
    const filteredTargets = getFilteredAndSortedWildcardTargets();
    const filteredTargetIds = new Set(filteredTargets.map(target => target.id));
    const newSelectedTargets = new Set([...selectedWildcardTargets].filter(targetId => !filteredTargetIds.has(targetId)));
    setSelectedWildcardTargets(newSelectedTargets);
  };

  const handleSelectAll = () => {
    const filteredTargets = getFilteredAndSortedWildcardTargets();
    const allTargetIds = filteredTargets.map(target => target.id);
    setSelectedWildcardTargets(new Set(allTargetIds));
  };

  const handleDeselectAll = () => {
    setSelectedWildcardTargets(new Set());
  };

  const getFilteredAndSortedWildcardTargets = () => {
    let filteredTargets = wildcardTargetsWithCounts.filter(target => {
      if (!target.scope_target) return false;
      
      if (filters.domain && !target.scope_target.toLowerCase().includes(filters.domain.toLowerCase())) {
        return false;
      }
      
      return true;
    });

    return filteredTargets.sort((a, b) => a.scope_target.localeCompare(b.scope_target));
  };

  const handleCloseModal = () => {
    setError('');
    setIsDragging(false);
    setDragStartIndex(null);
    setDragMode('select');
    handleClose();
  };

  const filteredWildcardTargets = getFilteredAndSortedWildcardTargets();
  const totalLiveWebServers = filteredWildcardTargets
    .filter(target => selectedWildcardTargets.has(target.id))
    .reduce((sum, target) => sum + target.liveWebServersCount, 0);

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
          <i className="bi bi-search me-2" />
          Configure DNSx - DNS Enumeration
        </Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {error && (
          <Alert variant="danger" dismissible onClose={() => setError('')}>
            {error}
          </Alert>
        )}

        <div className="mb-4">
          <Alert variant="info">
            <div className="d-flex align-items-center">
              <i className="bi bi-info-circle-fill me-2" />
              <div>
                <strong>DNSx Configuration:</strong> Select wildcard targets to scan with DNSx for DNS enumeration and cloud provider detection.
                Selected targets: <strong>{selectedWildcardTargets.size}</strong> 
                | Total FQDNs to scan: <strong>{totalLiveWebServers}</strong>
                | Estimated time: <strong>{estimatedTime === 1 ? '~1 hour' : `~${estimatedTime} hours`}</strong>
              </div>
            </div>
          </Alert>
        </div>

        {wildcardTargetsWithCounts.length === 0 ? (
          <div className="text-center py-4">
            {loadingCounts ? (
              <>
                <div className="spinner-border text-danger mb-3" role="status">
                  <span className="visually-hidden">Loading...</span>
                </div>
                <h5 className="text-white-50">Loading Wildcard Targets...</h5>
                <p className="text-white-50">
                  Fetching wildcard targets and live web server counts...
                </p>
              </>
            ) : (
              <>
                <i className="bi bi-diagram-3 text-white-50" style={{ fontSize: '3rem' }} />
                <h5 className="text-white-50 mt-3">No Wildcard Targets Available</h5>
                <p className="text-white-50">
                  Create wildcard targets from consolidated root domains first.
                </p>
              </>
            )}
          </div>
        ) : (
          <>
            <div className="d-flex justify-content-between align-items-center mb-3">
              <h6 className="mb-0 text-white">
                Select wildcard targets for DNSx scanning 
                <span className="text-light ms-2">({selectedWildcardTargets.size}/{wildcardTargetsWithCounts.length})</span>
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
                disabled={selectedWildcardTargets.size === 0}
                style={{ flex: 1 }}
              >
                <FaTimes className="me-1" />
                De-Select All
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={handleSelectAll}
                disabled={filteredWildcardTargets.length === 0}
                style={{ flex: 1 }}
              >
                <FaCheck className="me-1" />
                Select All Filtered
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={selectAllFiltered}
                disabled={filteredWildcardTargets.length === 0}
                style={{ flex: 1 }}
              >
                <FaCheck className="me-1" />
                Select All Visible
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={deselectAllFiltered}
                disabled={selectedWildcardTargets.size === 0}
                style={{ flex: 1 }}
              >
                <FaTimes className="me-1" />
                Deselect All Visible
              </Button>
            </div>

            <div className="mb-3">
              <small className="text-white-50">
                Showing {filteredWildcardTargets.length} of {wildcardTargetsWithCounts.length} wildcard targets
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
                        checked={filteredWildcardTargets.length > 0 && filteredWildcardTargets.every(target => {
                          return selectedWildcardTargets.has(target.id);
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
                    <th style={{ backgroundColor: 'var(--bs-dark)' }}>Wildcard Domain</th>
                    <th style={{ backgroundColor: 'var(--bs-dark)' }} className="text-center">Live Web Servers</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredWildcardTargets.map((target, index) => {
                    const isSelected = selectedWildcardTargets.has(target.id);
                    
                    return (
                      <tr 
                        key={target.id}
                        style={{
                          backgroundColor: isSelected 
                            ? 'rgba(220, 53, 69, 0.25)' 
                            : 'transparent',
                          cursor: 'pointer',
                          userSelect: 'none',
                          transition: 'background-color 0.15s ease-in-out'
                        }}
                        onMouseDown={(e) => handleMouseDown(target.id, index, e)}
                        onMouseEnter={() => handleMouseEnter(target.id, index)}
                      >
                        <td>
                          <Form.Check
                            type="checkbox"
                            checked={isSelected}
                            onChange={() => handleWildcardTargetSelect(target.id, index)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </td>
                        <td 
                          style={{ 
                            fontFamily: 'monospace',
                            fontSize: '0.875rem'
                          }}
                        >
                          {target.scope_target}
                        </td>
                        <td className="text-center">
                          <span className="badge bg-secondary">
                            {target.liveWebServersCount}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </Table>
            </div>

            {filteredWildcardTargets.length === 0 && (
              <div className="text-center py-4">
                <i className="bi bi-funnel text-white-50" style={{ fontSize: '2rem' }} />
                <h6 className="text-white-50 mt-2">No wildcard targets match the current filters</h6>
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
            {selectedWildcardTargets.size > 0 && (
              <>
                <i className="bi bi-clock me-1" />
                Total FQDNs to scan: {totalLiveWebServers} | Estimated time: {estimatedTime === 1 ? '~1 hour' : `~${estimatedTime} hours`}
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
              disabled={saving || selectedWildcardTargets.size === 0}
            >
              {saving ? (
                <>
                  <Spinner animation="border" size="sm" className="me-2" />
                  Saving...
                </>
              ) : (
                <>
                  <i className="bi bi-save me-2" />
                  Save Configuration ({selectedWildcardTargets.size} targets)
                </>
              )}
            </Button>
          </div>
        </div>
      </Modal.Footer>
    </Modal>
  );
};

export default DNSxConfigModal; 
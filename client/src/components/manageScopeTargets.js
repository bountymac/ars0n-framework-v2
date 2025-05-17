import { Row, Col, Button, Card, Alert, Spinner, ProgressBar } from 'react-bootstrap';
import { useState, useEffect, useRef } from 'react';
import AutoScanConfigModal from '../modals/autoScanConfigModal';
import { getHttpxResultsCount } from '../utils/miscUtils';

function ManageScopeTargets({ 
  handleOpen, 
  handleActiveModalOpen, 
  activeTarget, 
  scopeTargets, 
  getTypeIcon,
  onAutoScan,
  isAutoScanning,
  isAutoScanPaused,
  isAutoScanPausing,
  isAutoScanCancelling,
  setIsAutoScanPausing,
  setIsAutoScanCancelling,
  autoScanCurrentStep,
  mostRecentGauScanStatus,
  consolidatedSubdomains = [],
  mostRecentHttpxScan,
  onOpenAutoScanHistory
}) {
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [autoScanConfig, setAutoScanConfig] = useState(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [scanStartTime, setScanStartTime] = useState(null);
  const [scanEndTime, setScanEndTime] = useState(null);
  const [elapsed, setElapsed] = useState('');
  const [finalDuration, setFinalDuration] = useState('');
  const prevIsAutoScanning = useRef(isAutoScanning);
  const intervalRef = useRef(null);
  const [displayStatus, setDisplayStatus] = useState('idle');
  const resetTimeoutRef = useRef(null);
  const [isResuming, setIsResuming] = useState(false);

  useEffect(() => {
    const fetchConfig = async () => {
      setConfigLoading(true);
      try {
        const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`);
        if (response.ok) {
          const data = await response.json();
          setAutoScanConfig(data);
          console.log('[AutoScanConfig] Fetched from backend:', data);
        }
      } catch (e) {
        const fallback = {
          amass: true, sublist3r: true, assetfinder: true, gau: true, ctl: true, subfinder: true, consolidate_httpx_round1: true, shuffledns: true, cewl: true, consolidate_httpx_round2: true, gospider: true, subdomainizer: true, consolidate_httpx_round3: true, nuclei_screenshot: true, metadata: true, maxConsolidatedSubdomains: 2500, maxLiveWebServers: 500
        };
        setAutoScanConfig(fallback);
        console.log('[AutoScanConfig] Fallback to defaults:', fallback);
      } finally {
        setConfigLoading(false);
      }
    };
    fetchConfig();
  }, []);

  useEffect(() => {
    if (!prevIsAutoScanning.current && isAutoScanning) {
      // Scan starting
      setScanStartTime(new Date());
      setScanEndTime(null);
      setFinalDuration('');
      setDisplayStatus('running');
      
      // Clear any pending reset timeout
      if (resetTimeoutRef.current) {
        clearTimeout(resetTimeoutRef.current);
        resetTimeoutRef.current = null;
      }
    } else if (prevIsAutoScanning.current && !isAutoScanning) {
      // Scan completing
      setScanEndTime(new Date());
      setDisplayStatus('completed');
      
      if (scanStartTime) {
        const now = new Date();
        const diff = now - new Date(scanStartTime);
        const mins = Math.floor(diff / 60000);
        const secs = Math.floor((diff % 60000) / 1000);
        setFinalDuration(`${mins}m ${secs < 10 ? '0' : ''}${secs}s`);
      }
      
      // Set a timeout to reset status to idle after 5 seconds
      resetTimeoutRef.current = setTimeout(() => {
        setDisplayStatus('idle');
        console.log('Reset to idle status after 5-second delay');
      }, 5000);
    }
    
    // Update displayStatus based on pause state
    if (isAutoScanning && isAutoScanPaused) {
      setDisplayStatus('paused');
    } else if (isAutoScanning && !isAutoScanPaused) {
      setDisplayStatus('running');
    }
    
    prevIsAutoScanning.current = isAutoScanning;
  }, [isAutoScanning, scanStartTime, isAutoScanPaused]);

  // Clean up timeout on unmount
  useEffect(() => {
    return () => {
      if (resetTimeoutRef.current) {
        clearTimeout(resetTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (isAutoScanning && scanStartTime) {
      intervalRef.current = setInterval(() => {
        const now = scanEndTime ? new Date(scanEndTime) : new Date();
        const diff = now - new Date(scanStartTime);
        const mins = Math.floor(diff / 60000);
        const secs = Math.floor((diff % 60000) / 1000);
        setElapsed(`${mins}m ${secs < 10 ? '0' : ''}${secs}s`);
      }, 1000);
      return () => clearInterval(intervalRef.current);
    } else {
      setElapsed('');
      clearInterval(intervalRef.current);
    }
  }, [isAutoScanning, scanStartTime, scanEndTime]);

  const handleConfigure = async () => {
    // Fetch latest config before showing the modal
    setConfigLoading(true);
    try {
      const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`);
      if (response.ok) {
        const data = await response.json();
        setAutoScanConfig(data);
        console.log('[AutoScanConfig] Fetched fresh config before opening modal:', data);
      }
    } catch (e) {
      console.error('[AutoScanConfig] Error fetching config:', e);
    } finally {
      setConfigLoading(false);
      setShowConfigModal(true);
    }
  };

  const handleConfigSave = async (config) => {
    setConfigLoading(true);
    console.log('[AutoScanConfig] Saving config to backend:', config);
    try {
      const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      });
      if (response.ok) {
        const data = await response.json();
        setAutoScanConfig(data);
        setShowConfigModal(false);
        console.log('[AutoScanConfig] Saved and updated config from backend:', data);
      }
    } finally {
      setConfigLoading(false);
    }
  };

  const handlePause = async () => {
    if (!activeTarget || !activeTarget.id) return;
    
    if (!isAutoScanPaused) {
      // Pause the scan
      setIsAutoScanPausing(true);
      try {
        const response = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
              current_step: autoScanCurrentStep,
              is_paused: true,
              is_cancelled: false
            })
          }
        );
        if (!response.ok) {
          console.error('Error pausing auto scan:', await response.text());
          setIsAutoScanPausing(false);
        }
      } catch (error) {
        console.error('Error pausing auto scan:', error);
        setIsAutoScanPausing(false);
      }
    } else {
      // Resume the scan - set resuming state immediately
      setIsResuming(true);
      try {
        const response = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
              current_step: autoScanCurrentStep,
              is_paused: false,
              is_cancelled: false
            })
          }
        );
        if (!response.ok) {
          console.error('Error resuming auto scan:', await response.text());
          setIsResuming(false);
        }
        
        // Poll to detect when the scan actually resumes
        const checkInterval = setInterval(async () => {
          try {
            const statusResponse = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
            );
            
            if (statusResponse.ok) {
              const data = await statusResponse.json();
              if (!data.is_paused) {
                setIsResuming(false);
                clearInterval(checkInterval);
              }
            }
          } catch (error) {
            console.error('Error checking auto scan state:', error);
          }
        }, 1000);
        
        // Clear the interval after 10 seconds maximum
        setTimeout(() => {
          clearInterval(checkInterval);
          setIsResuming(false); // Reset resuming state after timeout
        }, 10000);
      } catch (error) {
        console.error('Error resuming auto scan:', error);
        setIsResuming(false);
      }
    }
  };

  const handleCancel = async () => {
    if (!activeTarget || !activeTarget.id) return;
    
    setIsAutoScanCancelling(true);
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ 
            current_step: autoScanCurrentStep,
            is_paused: false,
            is_cancelled: true
          })
        }
      );
      
      if (!response.ok) {
        console.error('Error cancelling auto scan:', await response.text());
        setIsAutoScanCancelling(false);
      } else {
        // Poll status to check if scan has already completed
        const checkInterval = setInterval(async () => {
          try {
            const statusResponse = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
            );
            
            if (statusResponse.ok) {
              const data = await statusResponse.json();
              if (data.current_step === 'completed' || !isAutoScanning) {
                setIsAutoScanCancelling(false);
                clearInterval(checkInterval);
              }
            }
          } catch (error) {
            console.error('Error checking auto scan state:', error);
          }
        }, 1000);
        
        // Clear the interval after 10 seconds maximum
        setTimeout(() => {
          clearInterval(checkInterval);
        }, 10000);
      }
    } catch (error) {
      console.error('Error cancelling auto scan:', error);
      setIsAutoScanCancelling(false);
    }
  };

  const formatStepName = (stepKey) => {
    if (!stepKey) return "";
    if (stepKey === 'gau' && mostRecentGauScanStatus === 'processing') {
      return "GAU (Processing Large Results)";
    }
    const words = stepKey
      .replace(/([A-Z])/g, ' $1')
      .replace(/_/g, ' ')
      .toLowerCase()
      .split(' ')
      .filter(word => word.length > 0)
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
    return words;
  };

  const getAutoScanPhase = (step) => {
    if (!step || step === 'idle' || step === 'completed') return "";
    
    // Define phases based on step groups
    if (['amass', 'sublist3r', 'assetfinder', 'gau', 'ctl', 'subfinder'].includes(step)) {
      return "Phase 1: Initial Subdomain Discovery";
    } else if (step.includes('consolidate_httpx_round1')) {
      return "Phase 2: Consolidating Initial Results";
    } else if (['shuffledns', 'cewl'].includes(step)) {
      return "Phase 3: Brute Force Discovery";
    } else if (step.includes('consolidate_httpx_round2')) {
      return "Phase 4: Consolidating Brute Force Results";
    } else if (['gospider', 'subdomainizer'].includes(step)) {
      return "Phase 5: JavaScript/Link Discovery";
    } else if (step.includes('consolidate_httpx_round3')) {
      return "Phase 6: Final Consolidation";
    } else if (['nuclei_screenshot', 'metadata'].includes(step)) {
      return "Phase 7: Target Analysis";
    }
    
    return "";
  };

  const getAutoScanStatusMessage = (step) => {
    if (!step || step === 'idle') return "Ready to start";
    if (step === 'completed') return "Scan completed";
    if (isAutoScanPaused) return "Scan paused";
    
    const stepName = formatStepName(step);
    
    if (step.includes('consolidate')) {
      return `Consolidating subdomains and discovering live web servers`;
    } else if (step === 'nuclei_screenshot') {
      return `Taking screenshots of discovered web servers`;
    } else if (step === 'metadata') {
      return `Gathering metadata from discovered web servers`;
    } else {
      return `Running ${stepName} scan`;
    }
  };

  const calculateProgress = () => {
    // If the display status is idle, always return 0
    if (displayStatus === 'idle' || !autoScanConfig || !autoScanCurrentStep || autoScanCurrentStep === 'idle') return 0;
    
    // If the display status is completed, return 100 for 5 seconds before resetting
    if (displayStatus === 'completed') return 100;
    
    // List of all possible steps in order, treating consolidation+httpx as single steps
    const stepOrder = [
      'amass', 'sublist3r', 'assetfinder', 'gau', 'ctl', 'subfinder',
      'consolidate_httpx_round1',
      'shuffledns', 'cewl',
      'consolidate_httpx_round2',
      'gospider', 'subdomainizer',
      'consolidate_httpx_round3',
      'nuclei_screenshot', 'metadata'
    ];
    
    // Enabled steps
    const enabledSteps = stepOrder.filter(key => autoScanConfig[key] !== false);
    
    // Current step index - handle both consolidation and httpx steps
    let currentIndex = -1;
    for (let i = 0; i < enabledSteps.length; i++) {
      const step = enabledSteps[i];
      if (step === autoScanCurrentStep) {
        currentIndex = i;
        break;
      } else if (step.includes('consolidate_httpx') && 
                (autoScanCurrentStep.includes('consolidate') || autoScanCurrentStep === 'httpx')) {
        currentIndex = i;
        break;
      }
    }
    
    if (currentIndex === -1) return 0;
    
    // Calculate progress percentage - cap at 95% until completed
    const progress = Math.round(((currentIndex + 1) / enabledSteps.length) * 100);
    return Math.min(progress, 95);
  };

  return (
    <>
      <Row className="mb-3">
        <Col>
          <h3 className="text-secondary">Active Scope Target</h3>
        </Col>
        <Col className="text-end">
          <Button variant="outline-danger" onClick={handleOpen}>
            Add Scope Target
          </Button>
          <Button variant="outline-danger" onClick={handleActiveModalOpen} className="ms-2">
            Select Active Target
          </Button>
        </Col>
      </Row>
      <Row className="mb-3">
        <Col>
          {activeTarget && (
            <Card variant="outline-danger">
              <Card.Body>
                <Card.Text className="d-flex justify-content-between text-danger">
                  <span style={{ fontSize: '22px' }}>
                    <strong>{activeTarget.scope_target}</strong>
                  </span>
                  <span>
                    <img src={getTypeIcon(activeTarget.type)} alt={activeTarget.type} style={{ width: '30px' }} />
                  </span>
                </Card.Text>
                {/* Auto Scan Status Section */}
                <div className="mb-3">
                  <div className="d-flex justify-content-between align-items-center mb-1 w-100">
                    <div className="d-flex flex-column">
                      <div className="d-flex align-items-center mb-1">
                        <span className={`fw-bold text-${displayStatus === 'running' ? 'danger' : displayStatus === 'completed' ? 'success' : 'secondary'}`}>
                          Auto Scan Status: {displayStatus === 'running' ? 'Running' : displayStatus === 'completed' ? 'Completed' : 'Idle'}
                        </span>
                        {displayStatus === 'running' && <Spinner animation="border" size="sm" variant="danger" className="ms-2" />}
                      </div>
                      <div className="mb-1">
                        <span className="text-white-50">Start Time: </span>
                        <span className="text-white">{scanStartTime ? new Date(scanStartTime).toLocaleTimeString() : '--:--:-- --'}</span>
                      </div>
                      {isAutoScanning ? (
                        <div className="mb-1">
                          <span className="text-white-50">Elapsed: </span>
                          <span className="text-white">{elapsed || '0m 00s'}</span>
                        </div>
                      ) : (
                        <div className="mb-1">
                          <span className="text-white-50">Duration: </span>
                          <span className="text-white">{finalDuration || (scanEndTime ? '0m 00s' : '--')}</span>
                        </div>
                      )}
                    </div>
                    <div className="text-end" style={{ minWidth: 180 }}>
                      <div className="text-white-50 small mb-2">
                        Consolidated Subdomains: {consolidatedSubdomains.length} / {autoScanConfig?.maxConsolidatedSubdomains ?? 2500}
                      </div>
                      <div className="text-white-50 small mb-2">
                        Live Web Servers: {getHttpxResultsCount(mostRecentHttpxScan)} / {autoScanConfig?.maxLiveWebServers ?? 500}
                      </div>
                    </div>
                  </div>
                  
                  {/* Auto Scan Status - Always shown */}
                  <div className="mt-3">
                    <div className="d-flex justify-content-between align-items-center mb-2">
                      <div className="text-white-50 small">
                        {displayStatus === 'running' ? (
                          <>
                            <span className="text-danger">●</span> Running {formatStepName(autoScanCurrentStep)}
                          </>
                        ) : displayStatus === 'completed' ? (
                          <>
                            <span className="text-success">●</span> Scan completed
                          </>
                        ) : (
                          <>
                            <span className="text-secondary">●</span> Ready to scan
                          </>
                        )}
                      </div>
                      <div className="text-white-50 small">
                        {scanEndTime && (
                          <>
                            Duration: {finalDuration}
                          </>
                        )}
                      </div>
                    </div>

                    {/* Progress Bar - Always shown */}
                    <div className="mt-2">
                      <div className="d-flex justify-content-between mb-1">
                        <span className="text-white-50 small">Progress</span>
                        <span className="text-white small">
                          {displayStatus === 'idle' ? '0' : calculateProgress()}%
                        </span>
                      </div>
                      <ProgressBar 
                        now={calculateProgress()} 
                        variant="danger" 
                        className="bg-dark" 
                        style={{ height: '8px' }}
                      />
                    </div>
                  </div>
                </div>
                <div className="d-flex justify-content-between gap-2 mt-3">
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={onOpenAutoScanHistory}
                  >
                    Scan History
                  </Button>
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={handleConfigure}
                  >
                    Configure
                  </Button>
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={onAutoScan}
                    disabled={isAutoScanning}
                  >
                    <div className="btn-content">
                      {isAutoScanning ? (
                        <Spinner animation="border" size="sm" variant="danger" />
                      ) : 'Auto Scan'}
                    </div>
                  </Button>
                  {isAutoScanPaused ? (
                    <Button 
                      variant="outline-danger" 
                      className="flex-fill" 
                      onClick={handlePause}
                      disabled={!isAutoScanning || isResuming}
                    >
                      {isResuming ? (
                        <div className="btn-content">
                          <span className="me-1">Resuming</span>
                          <Spinner animation="border" size="sm" />
                        </div>
                      ) : 'Resume'}
                    </Button>
                  ) : (
                    <Button 
                      variant="outline-danger" 
                      className="flex-fill" 
                      onClick={handlePause}
                      disabled={!isAutoScanning || isAutoScanCancelling}
                    >
                      {isAutoScanPausing ? (
                        <div className="btn-content">
                          <span className="me-1">Pausing</span>
                          <Spinner animation="border" size="sm" />
                        </div>
                      ) : 'Pause'}
                    </Button>
                  )}
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={handleCancel}
                    disabled={!isAutoScanning || isAutoScanPaused}
                  >
                    {isAutoScanCancelling ? (
                      <div className="btn-content">
                        <span className="me-1">Cancelling</span>
                        <Spinner animation="border" size="sm" />
                      </div>
                    ) : 'Cancel'}
                  </Button>
                </div>
              </Card.Body>
            </Card>
          )}
        </Col>
      </Row>
      {scopeTargets.length === 0 && (
        <Alert variant="danger" className="mt-3">
          No scope targets available. Please add a new target.
        </Alert>
      )}

      <AutoScanConfigModal
        show={showConfigModal}
        handleClose={() => setShowConfigModal(false)}
        onSave={handleConfigSave}
        config={autoScanConfig}
        loading={configLoading}
      />
    </>
  );
}

export default ManageScopeTargets;

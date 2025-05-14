import { Row, Col, Button, Card, Alert, Spinner } from 'react-bootstrap';
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
  autoScanCurrentStep,
  mostRecentGauScanStatus,
  consolidatedSubdomains = [],
  mostRecentHttpxScan
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
      setScanStartTime(new Date());
      setScanEndTime(null);
      setFinalDuration('');
    } else if (prevIsAutoScanning.current && !isAutoScanning) {
      setScanEndTime(new Date());
      if (scanStartTime) {
        const now = new Date();
        const diff = now - new Date(scanStartTime);
        const mins = Math.floor(diff / 60000);
        const secs = Math.floor((diff % 60000) / 1000);
        setFinalDuration(`${mins}m ${secs < 10 ? '0' : ''}${secs}s`);
      }
    }
    prevIsAutoScanning.current = isAutoScanning;
  }, [isAutoScanning]);

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

  const handleConfigure = () => {
    setShowConfigModal(true);
    console.log('[AutoScanConfig] Modal opened. Current config:', autoScanConfig);
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

  const handlePause = () => {
    console.log('Pause button clicked');
  };

  const handleCancel = () => {
    console.log('Cancel button clicked');
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
                        <span className={`fw-bold text-${isAutoScanning ? 'danger' : scanEndTime ? 'success' : 'secondary'}`}>Auto Scan Status: {isAutoScanning ? 'Running' : scanEndTime ? 'Completed' : 'Idle'}</span>
                        {isAutoScanning && <Spinner animation="border" size="sm" variant="danger" className="ms-2" />}
                      </div>
                      {isAutoScanning && autoScanCurrentStep && autoScanCurrentStep !== 'idle' && autoScanCurrentStep !== 'completed' && (
                        <div className="mb-1">
                          <span className="text-white-50">Current Step: </span>
                          <span className="text-white">{formatStepName(autoScanCurrentStep)}</span>
                        </div>
                      )}
                      {scanStartTime && (
                        <div className="mb-1">
                          <span className="text-white-50">Start Time: </span>
                          <span className="text-white">{new Date(scanStartTime).toLocaleTimeString()}</span>
                        </div>
                      )}
                      {isAutoScanning && (
                        <div className="mb-1">
                          <span className="text-white-50">Elapsed: </span>
                          <span className="text-white">{elapsed}</span>
                        </div>
                      )}
                      {!isAutoScanning && scanEndTime && scanStartTime && (
                        <div className="mb-1">
                          <span className="text-white-50">Duration: </span>
                          <span className="text-white">{finalDuration}</span>
                        </div>
                      )}
                    </div>
                    <div className="text-end" style={{ minWidth: 180 }}>
                      {/* Tools Run Count */}
                      {autoScanConfig && (
                        (() => {
                          // List of all possible steps in order
                          const stepOrder = [
                            'amass', 'sublist3r', 'assetfinder', 'gau', 'ctl', 'subfinder',
                            'consolidate_httpx_round1', 'shuffledns', 'cewl', 'consolidate_httpx_round2',
                            'gospider', 'subdomainizer', 'consolidate_httpx_round3', 'nuclei_screenshot', 'metadata'
                          ];
                          // Enabled steps
                          const enabledSteps = stepOrder.filter(key => autoScanConfig[key] !== false);
                          // Current step index
                          let runCount = 0;
                          if (isAutoScanning && autoScanCurrentStep && autoScanCurrentStep !== 'idle' && autoScanCurrentStep !== 'completed') {
                            const idx = enabledSteps.findIndex(key => formatStepName(key) === formatStepName(autoScanCurrentStep));
                            runCount = idx >= 0 ? idx + 1 : 0;
                          } else if (!isAutoScanning && scanEndTime) {
                            runCount = enabledSteps.length;
                          }
                          return (
                            <div className="text-white-50 small mb-1">Tools Run: <span className="text-white">{runCount}</span> / <span className="text-white">{enabledSteps.length}</span></div>
                          );
                        })()
                      )}
                      <div className="text-white-50 small">Consolidated Subdomains: <span className="text-white">{consolidatedSubdomains.length}</span> / <span className="text-white">{autoScanConfig?.maxConsolidatedSubdomains ?? 2500}</span></div>
                      <div className="text-white-50 small">Live Web Servers: <span className="text-white">{getHttpxResultsCount ? getHttpxResultsCount(mostRecentHttpxScan) : 0}</span> / <span className="text-white">{autoScanConfig?.maxLiveWebServers ?? 500}</span></div>
                    </div>
                  </div>
                </div>
                <div className="d-flex justify-content-between gap-2 mt-3">
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
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={handlePause}
                    disabled={!isAutoScanning}
                  >
                    Pause
                  </Button>
                  <Button 
                    variant="outline-danger" 
                    className="flex-fill" 
                    onClick={handleCancel}
                    disabled={!isAutoScanning}
                  >
                    Cancel
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

import { Row, Col, Button, Card, Alert } from 'react-bootstrap';
import { useState, useEffect } from 'react';
import AutoScanConfigModal from '../modals/autoScanConfigModal';

function ManageScopeTargets({ 
  handleOpen, 
  handleActiveModalOpen, 
  activeTarget, 
  scopeTargets, 
  getTypeIcon,
  onAutoScan,
  isAutoScanning,
  autoScanCurrentStep,
  mostRecentGauScanStatus
}) {
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [autoScanConfig, setAutoScanConfig] = useState(null);
  const [configLoading, setConfigLoading] = useState(true);

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
        // fallback to defaults if needed
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

  // Helper function to display a human-readable step name
  const formatStepName = (stepKey) => {
    if (!stepKey) return "";
    
    // Special case for GAU when it's processing
    if (stepKey === 'gau' && mostRecentGauScanStatus === 'processing') {
      return "GAU (Processing Large Results)";
    }
    
    // Convert snake_case or camelCase to words with spaces
    const words = stepKey
      .replace(/([A-Z])/g, ' $1') // Insert space before capital letters
      .replace(/_/g, ' ') // Replace underscores with spaces
      .toLowerCase()
      .split(' ')
      .filter(word => word.length > 0)
      .map(word => word.charAt(0).toUpperCase() + word.slice(1)) // Capitalize first letter
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
                        <>
                          <div className="spinner"></div>
                          {autoScanCurrentStep && autoScanCurrentStep !== 'idle' && autoScanCurrentStep !== 'completed' && (
                            <span className="ms-2">{formatStepName(autoScanCurrentStep)}</span>
                          )}
                        </>
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

import { useState, useEffect } from 'react';
import AddScopeTargetModal from './modals/addScopeTargetModal.js';
import SelectActiveScopeTargetModal from './modals/selectActiveScopeTargetModal.js';
import { DNSRecordsModal, SubdomainsModal, CloudDomainsModal, InfrastructureMapModal } from './modals/amassModals.js';
import Ars0nFrameworkHeader from './components/ars0nFrameworkHeader.js';
import ManageScopeTargets from './components/manageScopeTargets.js';
import fetchAmassScans from './utils/fetchAmassScans.js';
import {
    Container,
    Fade,
    Card,
    Row,
    Col,
    Button,
    ListGroup,
    Accordion,
    Modal,
    Table,
    Toast,
    ToastContainer,
} from 'react-bootstrap';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import initiateAmassScan from './utils/initiateAmassScan';
import monitorScanStatus from './utils/monitorScanStatus';
import validateInput from './utils/validateInput.js';
import {
    getTypeIcon,
    getModeIcon,
    getLastScanDate,
    getLatestScanStatus,
    getLatestScanTime,
    getLatestScanId,
    getExecutionTime,
    getResultLength,
    copyToClipboard,
} from './utils/miscUtils.js';
import { MdCopyAll, MdCheckCircle } from 'react-icons/md';

function App() {
  const [showScanHistoryModal, setShowScanHistoryModal] = useState(false);
  const [showRawResultsModal, setShowRawResultsModal] = useState(false);
  const [showDNSRecordsModal, setShowDNSRecordsModal] = useState(false);
  const [scanHistory, setScanHistory] = useState([]);
  const [rawResults, setRawResults] = useState([]);
  const [dnsRecords, setDnsRecords] = useState([]);
  const [showModal, setShowModal] = useState(false);
  const [showActiveModal, setShowActiveModal] = useState(false);
  const [selections, setSelections] = useState({
    type: '',
    mode: '',
    inputText: '',
  });
  const [scopeTargets, setScopeTargets] = useState([]);
  const [activeTarget, setActiveTarget] = useState(null);
  const [amassScans, setAmassScans] = useState([]);
  const [errorMessage, setErrorMessage] = useState('');
  const [fadeIn, setFadeIn] = useState(false);
  const [mostRecentAmassScanStatus, setMostRecentAmassScanStatus] = useState(null);
  const [mostRecentAmassScan, setMostRecentAmassScan] = useState(null);
  const [isScanning, setIsScanning] = useState(false);
  const [subdomains, setSubdomains] = useState([]);
  const [showSubdomainsModal, setShowSubdomainsModal] = useState(false);
  const [cloudDomains, setCloudDomains] = useState([]);
  const [showCloudDomainsModal, setShowCloudDomainsModal] = useState(false);
  const [showToast, setShowToast] = useState(false);
  const [showInfraModal, setShowInfraModal] = useState(false);

  const handleCloseSubdomainsModal = () => setShowSubdomainsModal(false);
  const handleCloseCloudDomainsModal = () => setShowCloudDomainsModal(false);

  useEffect(() => {
    fetchScopeTargets();
  }, [isScanning]);

  useEffect(() => {
    if (activeTarget && amassScans.length > 0) {
      setScanHistory(amassScans);
    }
  }, [activeTarget, amassScans, isScanning]);

  useEffect(() => {
    if (activeTarget) {
      fetchAmassScans(activeTarget, setAmassScans, setMostRecentAmassScan, setMostRecentAmassScanStatus, setDnsRecords, setSubdomains, setCloudDomains);
    }
  }, [activeTarget, isScanning]);  

  useEffect(() => {
    if (activeTarget) {
      monitorScanStatus(
        activeTarget,
        setAmassScans,
        setMostRecentAmassScan,
        setIsScanning,
        setMostRecentAmassScanStatus,
        setDnsRecords,
        setSubdomains,
        setCloudDomains
      );
    }
  }, [activeTarget]);

  // Open Modal Handlers

  const handleOpenScanHistoryModal = () => {
    setScanHistory(amassScans)
    setShowScanHistoryModal(true);
  };

  const handleOpenRawResultsModal = () => {
    if (amassScans.length > 0) {
      const mostRecentScan = amassScans.reduce((latest, scan) => {
        const scanDate = new Date(scan.created_at);
        return scanDate > new Date(latest.created_at) ? scan : latest;
      }, amassScans[0]);

      const rawResults = mostRecentScan.result ? mostRecentScan.result.split('\n') : [];
      setRawResults(rawResults);
      setShowRawResultsModal(true);
    } else {
      setShowRawResultsModal(true);
      console.warn("No scans available for raw results");
    }
  };

  const handleOpenSubdomainsModal = async () => {
    if (amassScans.length > 0) {
      const mostRecentScan = amassScans.reduce((latest, scan) => {
        const scanDate = new Date(scan.created_at);
        return scanDate > new Date(latest.created_at) ? scan : latest;
      }, amassScans[0]);

      try {
        const response = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass/${mostRecentScan.scan_id}/subdomain`
        );
        if (!response.ok) {
          throw new Error('Failed to fetch subdomains');
        }
        const subdomainsData = await response.json();
        console.log(subdomainsData);
        setSubdomains(subdomainsData);
        setShowSubdomainsModal(true);
      } catch (error) {
        setShowSubdomainsModal(true);
        console.error("Error fetching subdomains:", error);
      }
    } else {
      setShowSubdomainsModal(true);
      console.warn("No scans available for subdomains");
    }
  };

  const handleOpenCloudDomainsModal = async () => {
    if (amassScans.length > 0) {
      const mostRecentScan = amassScans.reduce((latest, scan) => {
        const scanDate = new Date(scan.created_at);
        return scanDate > new Date(latest.created_at) ? scan : latest;
      }, amassScans[0]);

      try {
        const response = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass/${mostRecentScan.scan_id}/cloud`
        );
        if (!response.ok) {
          throw new Error('Failed to fetch cloud domains');
        }
        const cloudData = await response.json();

        const formattedCloudDomains = [];
        if (cloudData.aws_domains) {
          formattedCloudDomains.push(...cloudData.aws_domains.map((name) => ({ type: 'AWS', name })));
        }
        if (cloudData.azure_domains) {
          formattedCloudDomains.push(...cloudData.azure_domains.map((name) => ({ type: 'Azure', name })));
        }
        if (cloudData.gcp_domains) {
          formattedCloudDomains.push(...cloudData.gcp_domains.map((name) => ({ type: 'GCP', name })));
        }

        setCloudDomains(formattedCloudDomains);
        setShowCloudDomainsModal(true);
      } catch (error) {
        setCloudDomains([]);
        setShowCloudDomainsModal(true);
        console.error("Error fetching cloud domains:", error);
      }
    } else {
      setCloudDomains([]);
      setShowCloudDomainsModal(true);
      console.warn("No scans available for cloud domains");
    }
  };

  const handleOpenDNSRecordsModal = async () => {
    if (amassScans.length > 0) {
      const mostRecentScan = amassScans.reduce((latest, scan) => {
        const scanDate = new Date(scan.created_at);
        return scanDate > new Date(latest.created_at) ? scan : latest;
      }, amassScans[0]);

      try {
        const response = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass/${mostRecentScan.scan_id}/dns`
        );
        if (!response.ok) {
          throw new Error('Failed to fetch DNS records');
        }
        const dnsData = await response.json();
        if (dnsData !== null) {
          setDnsRecords(dnsData);
        } else {
          setDnsRecords([]);
        }
        setShowDNSRecordsModal(true);
      } catch (error) {
        setShowDNSRecordsModal(true);
        console.error("Error fetching DNS records:", error);
      }
    } else {
      setShowDNSRecordsModal(true);
      console.warn("No scans available for DNS records");
    }
  };

  const handleClose = () => {
    setShowModal(false);
    setErrorMessage('');
  };

  const handleActiveModalClose = () => {
    setShowActiveModal(false);
  };

  const handleActiveModalOpen = () => {
    setShowActiveModal(true);
  };

  const handleOpen = () => {
    setSelections({ type: '', mode: '', inputText: '' });
    setShowModal(true);
  };

  const handleSubmit = async () => {
    if (!validateInput(selections, setErrorMessage)) {
      return;
    }

    if (selections.type === 'Wildcard' && !selections.inputText.startsWith('*.')) {
      setSelections((prev) => ({ ...prev, inputText: `*.${prev.inputText}` }));
    }

    if (selections.type && selections.mode && selections.inputText) {
      try {
        const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/add`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            type: selections.type,
            mode: selections.mode,
            scope_target: selections.inputText,
          }),
        });

        if (!response.ok) {
          throw new Error('Failed to add scope target');
        }

        setSelections({ type: '', mode: '', inputText: '' });
        setShowModal(false);
        fetchScopeTargets();
      } catch (error) {
        console.error('Error adding scope target:', error);
        setErrorMessage('Failed to add scope target');
      }
    } else {
      setErrorMessage('You forgot something...');
    }
  };

  const handleDelete = async () => {
    if (!activeTarget) return;

    try {
      const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/delete/${activeTarget.id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to delete scope target');
      }

      setScopeTargets((prev) => {
        const updatedTargets = prev.filter((target) => target.id !== activeTarget.id);
        const newActiveTarget = updatedTargets.length > 0 ? updatedTargets[0] : null;
        setActiveTarget(newActiveTarget);
        if (!newActiveTarget && showActiveModal) {
          setShowActiveModal(false);
          setShowModal(true);
        }
        return updatedTargets;
      });
    } catch (error) {
      console.error('Error deleting scope target:', error);
    }
  };

  const fetchScopeTargets = async () => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/read`
      );
      if (!response.ok) {
        throw new Error('Failed to fetch scope targets');
      }
      const data = await response.json();
      setScopeTargets(data || []);
      setFadeIn(true);
      if (data && data.length > 0) {
        setActiveTarget(data[0]);
      } else {
        setShowModal(true);
      }
    } catch (error) {
      console.error('Error fetching scope targets:', error);
      setScopeTargets([]);
    }
  };

  const handleActiveSelect = (target) => {
    setActiveTarget(target);
  };

  const handleSelect = (key, value) => {
    setSelections((prev) => ({ ...prev, [key]: value }));
    setErrorMessage('');
  };

  const handleCloseScanHistoryModal = () => setShowScanHistoryModal(false);
  const handleCloseRawResultsModal = () => setShowRawResultsModal(false);
  const handleCloseDNSRecordsModal = () => setShowDNSRecordsModal(false);


  const startAmassScan = () => {
    initiateAmassScan(activeTarget, monitorScanStatus, setIsScanning, setAmassScans, setMostRecentAmassScanStatus, setDnsRecords, setSubdomains, setCloudDomains, setMostRecentAmassScan)
  }

  const renderScanId = (scanId) => {
    if (scanId === 'No scans available' || scanId === 'No scan ID available') {
      return <span>{scanId}</span>;
    }
    
    const handleCopy = async () => {
      const success = await copyToClipboard(scanId);
      if (success) {
        setShowToast(true);
        setTimeout(() => setShowToast(false), 3000); // Hide after 3 seconds
      }
    };

    return (
      <span className="scan-id-container">
        {scanId}
        <button 
          onClick={handleCopy}
          className="copy-button"
          title="Copy Scan ID"
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            padding: '4px',
          }}
        >
          <MdCopyAll size={14} />
        </button>
      </span>
    );
  };

  const handleOpenInfraModal = () => setShowInfraModal(true);
  const handleCloseInfraModal = () => setShowInfraModal(false);

  return (
    <Container data-bs-theme="dark" className="App" style={{ padding: '20px' }}>
      <Ars0nFrameworkHeader />

      <ToastContainer 
        position="bottom-center"
        style={{ 
          position: 'fixed', 
          bottom: 20,
          left: '50%',
          transform: 'translateX(-50%)',
          zIndex: 1000,
          minWidth: '300px'
        }}
      >
        <Toast 
          show={showToast} 
          onClose={() => setShowToast(false)}
          className={`custom-toast ${!showToast ? 'hide' : ''}`}
          autohide
          delay={3000}
        >
          <Toast.Header>
            <MdCheckCircle 
              className="success-icon me-2" 
              size={20} 
              color="#ff0000"
            />
            <strong className="me-auto" style={{ 
              color: '#ff0000',
              fontSize: '0.95rem',
              letterSpacing: '0.5px'
            }}>
              Success
            </strong>
          </Toast.Header>
          <Toast.Body style={{ color: '#ffffff' }}>
            <div className="d-flex align-items-center">
              <span>Scan ID Copied to Clipboard</span>
            </div>
          </Toast.Body>
        </Toast>
      </ToastContainer>

      <AddScopeTargetModal
        show={showModal}
        handleClose={handleClose}
        selections={selections}
        handleSelect={handleSelect}
        handleFormSubmit={handleSubmit}
        errorMessage={errorMessage}
      />

      <SelectActiveScopeTargetModal
        showActiveModal={showActiveModal}
        handleActiveModalClose={handleActiveModalClose}
        scopeTargets={scopeTargets}
        activeTarget={activeTarget}
        handleActiveSelect={handleActiveSelect}
        handleDelete={handleDelete}
      />

      <Modal data-bs-theme="dark" show={showScanHistoryModal} onHide={handleCloseScanHistoryModal} size="xl">
        <Modal.Header closeButton>
          <Modal.Title className='text-danger'>Scan History</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Table striped bordered hover>
            <thead>
              <tr>
                <th>Scan ID</th>
                <th>Execution Time</th>
                <th>Number of Results</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {scanHistory.map((scan) => (
                <tr key={scan.scan_id}>
                  <td>{scan.scan_id || "ERROR"}</td>
                  <td>{getExecutionTime(scan.execution_time) || "---"}</td>
                  <td>{getResultLength(scan) || "---"}</td>
                  <td>{Date(scan.created_at) || "ERROR"}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Modal.Body>
      </Modal>

      <Modal data-bs-theme="dark" show={showRawResultsModal} onHide={handleCloseRawResultsModal} size="lg">
        <Modal.Header closeButton>
          <Modal.Title className='text-danger'>Raw Results</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <ListGroup>
            {rawResults.map((result, index) => (
              <ListGroup.Item key={index} className="text-white bg-dark">
                {result}
              </ListGroup.Item>
            ))}
          </ListGroup>
        </Modal.Body>
      </Modal>

      <DNSRecordsModal
        showDNSRecordsModal={showDNSRecordsModal}
        handleCloseDNSRecordsModal={handleCloseDNSRecordsModal}
        dnsRecords={dnsRecords}
      />

      <SubdomainsModal
        showSubdomainsModal={showSubdomainsModal}
        handleCloseSubdomainsModal={handleCloseSubdomainsModal}
        subdomains={subdomains}
      />

      <CloudDomainsModal
        showCloudDomainsModal={showCloudDomainsModal}
        handleCloseCloudDomainsModal={handleCloseCloudDomainsModal}
        cloudDomains={cloudDomains}
      />

      <InfrastructureMapModal
        showInfraModal={showInfraModal}
        handleCloseInfraModal={handleCloseInfraModal}
        scanId={getLatestScanId(amassScans)}
      />

      <Fade in={fadeIn}>
        <ManageScopeTargets
          handleOpen={handleOpen}
          handleActiveModalOpen={handleActiveModalOpen}
          activeTarget={activeTarget}
          scopeTargets={scopeTargets}
          getTypeIcon={getTypeIcon}
          getModeIcon={getModeIcon}
        />
      </Fade>

      {activeTarget && (
        <Fade className="mt-3" in={fadeIn}>
          <div>
            {activeTarget.type === 'Company' && (
              <div className="mb-4">
                <h3 className="text-danger">Company</h3>
                <Row>
                  <Col md={6}>
                    <Card className="mb-3 shadow-sm">
                      <Card.Body>
                        <Card.Title>Row 1, Column 1</Card.Title>
                        <Card.Text>Content for Row 1, Column 1.</Card.Text>
                      </Card.Body>
                    </Card>
                  </Col>
                  <Col md={6}>
                    <Card className="mb-3 shadow-sm">
                      <Card.Body>
                        <Card.Title>Row 1, Column 2</Card.Title>
                        <Card.Text>Content for Row 1, Column 2.</Card.Text>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <Row>
                  <Col md={12}>
                    <Card className="mb-3 shadow-sm">
                      <Card.Body>
                        <Card.Title>Row 2, Single Column</Card.Title>
                        <Card.Text>Content for Row 2, Single Column.</Card.Text>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
              </div>
            )}
            {(activeTarget.type === 'Wildcard' || activeTarget.type === 'Company') && (
              <div className="mb-4">
                <h3 className="text-danger mb-3">Wildcard</h3>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm" style={{ minHeight: '250px' }}>
                      <Card.Body className="d-flex flex-column justify-content-between">
                        <div>
                          <Card.Title className="text-danger fs-3 mb-3 text-center">
                            <a href="https://github.com/OWASP/Amass" className="text-danger text-decoration-none">
                              Amass Enum
                            </a>
                          </Card.Title>
                          <Card.Text className="text-white small fst-italic text-center">
                            A powerful subdomain enumeration and OSINT tool for in-depth reconnaissance.
                          </Card.Text>
                          <Card.Text className="text-white small d-flex justify-content-between">
                            <span>Last Scanned: &nbsp;&nbsp;{getLastScanDate(amassScans)}</span>
                            <span>Total Results: {getResultLength(scanHistory[scanHistory.length - 1]) || "N/a"}</span>
                          </Card.Text>
                          <Card.Text className="text-white small d-flex justify-content-between">
                            <span>Last Scan Status: &nbsp;&nbsp;{getLatestScanStatus(amassScans)}</span>
                            <span>Cloud Domains: {cloudDomains.length || "0"}</span>
                          </Card.Text>
                          <Card.Text className="text-white small d-flex justify-content-between">
                            <span>Scan Time: &nbsp;&nbsp;{getExecutionTime(getLatestScanTime(amassScans))}</span>
                            <span>Subdomains: {subdomains.length || "0"}</span>
                          </Card.Text>
                          <Card.Text className="text-white small d-flex justify-content-between mb-3">
                            <span>Scan ID: {renderScanId(getLatestScanId(amassScans))}</span>
                            <span>DNS Records: {dnsRecords.length}</span>
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between w-100 mt-3 gap-2">
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenScanHistoryModal}>&nbsp;&nbsp;&nbsp;Scan History&nbsp;&nbsp;&nbsp;</Button>
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenRawResultsModal}>&nbsp;&nbsp;&nbsp;Raw Results&nbsp;&nbsp;&nbsp;</Button>
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenInfraModal}>Infrastructure Map</Button>
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenDNSRecordsModal}>&nbsp;&nbsp;&nbsp;DNS Records&nbsp;&nbsp;&nbsp;</Button>
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenSubdomainsModal}>&nbsp;&nbsp;&nbsp;Subdomains&nbsp;&nbsp;&nbsp;</Button>
                          <Button variant="outline-danger" className="flex-fill" onClick={handleOpenCloudDomainsModal}>&nbsp;&nbsp;Cloud Domains&nbsp;&nbsp;</Button>
                          <Button
                            variant="outline-danger"
                            onClick={startAmassScan}
                            disabled={isScanning || mostRecentAmassScanStatus === "pending" ? true : false}
                          >
                            {isScanning || mostRecentAmassScanStatus === "pending" ? <span className="blinking">&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Scanning...&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span> : 'Scan ' + activeTarget.scope_target}
                          </Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <h4 className="text-secondary mb-3 fs-5">Subdomain Scraping</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="row-cols-5 g-3 mb-4">
                  {[
                    { name: 'Sublist3r', link: 'https://github.com/aboul3la/Sublist3r' },
                    { name: 'Assetfinder', link: 'https://github.com/tomnomnom/assetfinder' },
                    { name: 'GAU', link: 'https://github.com/lc/gau' },
                    { name: 'CTL', link: 'https://github.com/chromium/ctlog' },
                    { name: 'Subfinder', link: 'https://github.com/projectdiscovery/subfinder' }
                  ].map((tool, index) => (
                    <Col key={index}>
                      <Card className="shadow-sm h-100 text-center" style={{ minHeight: '250px' }}>
                        <Card.Body className="d-flex flex-column">
                          <Card.Title className="text-danger mb-3">
                            <a href={tool.link} className="text-danger text-decoration-none">
                              {tool.name}
                            </a>
                          </Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            A subdomain enumeration tool that uses OSINT techniques.
                          </Card.Text>
                          <div className="d-flex justify-content-between mt-auto gap-2">
                            <Button variant="outline-danger" className="flex-fill">Results</Button>
                            <Button variant="outline-danger" className="flex-fill">Scan</Button>
                          </div>
                        </Card.Body>
                      </Card>
                    </Col>
                  ))}
                </Row>
                <h4 className="text-secondary mb-3 fs-5">Brute-Force</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="justify-content-between mb-4">
                  {[
                    { name: 'ShuffleDNS', link: 'https://github.com/projectdiscovery/shuffledns' },
                    { name: 'CeWL', link: 'https://github.com/digininja/CeWL' }
                  ].map((tool, index) => (
                    <Col md={6} className="mb-4" key={index}>
                      <Card className="shadow-sm h-100 text-center" style={{ minHeight: '150px' }}>
                        <Card.Body className="d-flex flex-column">
                          <Card.Title className="text-danger mb-3">
                            <a href={tool.link} className="text-danger text-decoration-none">
                              {tool.name}
                            </a>
                          </Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            A subdomain resolver tool that utilizes massdns for resolving subdomains.
                          </Card.Text>
                          <div className="d-flex justify-content-between mt-auto gap-2">
                            <Button variant="outline-danger" className="flex-fill">Results</Button>
                            <Button variant="outline-danger" className="flex-fill">Scan</Button>
                          </div>
                        </Card.Body>
                      </Card>
                    </Col>
                  ))}
                </Row>
                <h4 className="text-secondary mb-3 fs-5">Consolidate Subdomains & Live Web Servers - Round 1</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm">
                      <Card.Body className="d-flex align-items-center justify-content-between">
                        <div className="d-flex flex-column">
                          <Card.Title className="text-danger fs-4 mb-2">Consolidate Subdomains</Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            Each tool has discovered a list of subdomains. Now, we need to consolidate those lists into a single list of unique subdomains.
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between gap-2">
                          <Button variant="outline-danger" className="flex-fill">Results</Button>
                          <Button variant="outline-danger" className="flex-fill">Consolidate</Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm">
                      <Card.Body className="d-flex align-items-center justify-content-between">
                        <div className="d-flex flex-column">
                          <Card.Title className="text-danger fs-4 mb-2">Live Web Servers</Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            Now that we have a list of unique subdomains, we will use{' '}
                            <a
                              href="https://github.com/projectdiscovery/httpx"
                              className="text-danger text-decoration-none"
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              httpx
                            </a>{' '}
                            by Project Discovery to identify which of those domains are pointing to live web servers.
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between gap-2">
                          <Button variant="outline-danger" className="flex-fill">Results</Button>
                          <Button variant="outline-danger" className="flex-fill">Scan</Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <h4 className="text-secondary mb-3 fs-5">JavaScript/Link Discovery</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="justify-content-between mb-4">
                  {[
                    { name: 'GoSpider', link: 'https://github.com/jaeles-project/gospider' },
                    { name: 'Subdomainizer', link: 'https://github.com/nsonaniya2010/SubDomainizer' }
                  ].map((tool, index) => (
                    <Col md={6} className="mb-4" key={index}>
                      <Card className="shadow-sm h-100 text-center" style={{ minHeight: '150px' }}>
                        <Card.Body className="d-flex flex-column">
                          <Card.Title className="text-danger mb-3">
                            <a href={tool.link} className="text-danger text-decoration-none">
                              {tool.name}
                            </a>
                          </Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            A fast web spider written in Go for web scraping and crawling.
                          </Card.Text>
                          <div className="d-flex justify-content-between mt-auto gap-2">
                            <Button variant="outline-danger" className="flex-fill">Results</Button>
                            <Button variant="outline-danger" className="flex-fill">Scan</Button>
                          </div>
                        </Card.Body>
                      </Card>
                    </Col>
                  ))}
                </Row>
                <h4 className="text-secondary mb-3 fs-5">Consolidate Subdomains & Live Web Servers - Round 2</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm">
                      <Card.Body className="d-flex align-items-center justify-content-between">
                        <div className="d-flex flex-column">
                          <Card.Title className="text-danger fs-4 mb-2">Consolidate Subdomains</Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            Each tool has discovered a list of subdomains. Now, we need to consolidate those lists into a single list of unique subdomains.
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between gap-2">
                          <Button variant="outline-danger" className="flex-fill">Results</Button>
                          <Button variant="outline-danger" className="flex-fill">Consolidate</Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm">
                      <Card.Body className="d-flex align-items-center justify-content-between">
                        <div className="d-flex flex-column">
                          <Card.Title className="text-danger fs-4 mb-2">Live Web Servers</Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            Now that we have a list of unique subdomains, we will use{' '}
                            <a
                              href="https://github.com/projectdiscovery/httpx"
                              className="text-danger text-decoration-none"
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              httpx
                            </a>{' '}
                            by Project Discovery to identify which of those domains are pointing to live web servers.
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between gap-2">
                          <Button variant="outline-danger" className="flex-fill">Results</Button>
                          <Button variant="outline-danger" className="flex-fill">Scan</Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
                <h4 className="text-secondary mb-3 fs-3 text-center">DECISION POINT</h4>
                <Accordion data-bs-theme="dark" className="mb-3">
                  <Accordion.Item eventKey="0">
                    <Accordion.Header className="fs-5">Help Me Learn!</Accordion.Header>
                    <Accordion.Body className="bg-dark">
                      <ListGroup as="ul" variant="flush">
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic one{' '}
                          <a href="https://example.com/topic1" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                          <ListGroup as="ul" variant="flush" className="mt-2">
                            <ListGroup.Item as="li" className="bg-dark text-white fst-italic">
                              Minor Topic one{' '}
                              <a href="https://example.com/minor-topic1" className="text-danger text-decoration-none">
                                Learn More
                              </a>
                            </ListGroup.Item>
                          </ListGroup>
                        </ListGroup.Item>
                        <ListGroup.Item as="li" className="bg-dark text-white">
                          Major learning topic two{' '}
                          <a href="https://example.com/topic2" className="text-danger text-decoration-none">
                            Learn More
                          </a>
                        </ListGroup.Item>
                      </ListGroup>
                    </Accordion.Body>
                  </Accordion.Item>
                </Accordion>
                <Row className="mb-4">
                  <Col>
                    <Card className="shadow-sm" style={{ minHeight: '250px' }}>
                      <Card.Body className="d-flex flex-column justify-content-between text-center">
                        <div>
                          <Card.Title className="text-danger fs-3 mb-3">Select Target URL</Card.Title>
                          <Card.Text className="text-white small fst-italic">
                            We now have a list of unique subdomains pointing to live web servers. The next step is to take screenshots of each web application and gather data to identify the target that will give us the greatest ROI as a bug bounty hunter. Focus on signs that the target may have vulnerabilities, may not be maintained, or offers a large attack surface.
                          </Card.Text>
                        </div>
                        <div className="d-flex justify-content-between w-100 mt-3 gap-2">
                          <Button variant="outline-danger" className="flex-fill">Take Screenshots</Button>
                          <Button variant="outline-danger" className="flex-fill">Gather Metadata</Button>
                          <Button variant="outline-danger" className="flex-fill">Generate Report</Button>
                          <Button variant="outline-danger" className="flex-fill">Select Target URL</Button>
                        </div>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
              </div>
            )}
            {(activeTarget.type === 'Company' ||
              activeTarget.type === 'Wildcard' ||
              activeTarget.type === 'URL') && (
                <div className="mb-4">
                  <h3 className="text-danger">URL</h3>
                  <Row>
                    <Col md={12}>
                      <Card className="mb-3 shadow-sm">
                        <Card.Body>
                          <Card.Title>Row 1, Single Column</Card.Title>
                          <Card.Text>Details about the URL go here.</Card.Text>
                        </Card.Body>
                      </Card>
                    </Col>
                  </Row>
                </div>
              )}
          </div>
        </Fade>
      )}
    </Container>
  );
}

export default App;

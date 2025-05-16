// Define the auto scan steps
const AUTO_SCAN_STEPS = {
  IDLE: 'idle', // 0
  AMASS: 'amass', // 1
  SUBLIST3R: 'sublist3r', // 2
  ASSETFINDER: 'assetfinder', // 3
  GAU: 'gau', // 4
  CTL: 'ctl', // 5
  SUBFINDER: 'subfinder', // 6
  CONSOLIDATE: 'consolidate', // 7
  HTTPX: 'httpx', // 8
  SHUFFLEDNS: 'shuffledns', // 9
  SHUFFLEDNS_CEWL: 'shuffledns_cewl', // 10
  CONSOLIDATE_ROUND2: 'consolidate_round2', // 10.5
  HTTPX_ROUND2: 'httpx_round2', // 10.75
  GOSPIDER: 'gospider', // 11
  SUBDOMAINIZER: 'subdomainizer', // 12
  CONSOLIDATE_ROUND3: 'consolidate_round3', // 12.5
  HTTPX_ROUND3: 'httpx_round3', // 12.75
  NUCLEI_SCREENSHOT: 'nuclei-screenshot', // 13
  METADATA: 'metadata', // 14
  COMPLETED: 'completed' // 15
};

// Debug utility function
const debugTrace = (message) => {
  console.log(`[Auto-Scan Debug] ${message}`);
};

// Utility function to update the auto scan state on the server
const updateAutoScanState = async (targetId, currentStep) => {
  try {
    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${targetId}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ current_step: currentStep }),
      }
    );
    
    if (!response.ok) {
      throw new Error(`Failed to update auto scan state: ${response.statusText}`);
    }
    
    debugTrace(`API updated: current_step=${currentStep}`);
    return true;
  } catch (error) {
    debugTrace(`Error updating auto scan state: ${error.message}`);
    return false;
  }
};

// Helper function to wait for a scan to complete
const waitForScanCompletion = async (scanType, targetId, setIsScanning, setMostRecentScanStatus, setMostRecentScan = null) => {
  debugTrace(`waitForScanCompletion started for ${scanType}`);

  return new Promise((resolve) => {
    let attempts = 0;
    const checkStatus = async () => {
      attempts++;
      debugTrace(`Checking ${scanType} scan status - attempt #${attempts}`);
      try {
        const url = `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${targetId}/scans/${scanType}`;
        debugTrace(`Fetching scan status from: ${url}`);
        const response = await fetch(url);
        if (!response.ok) {
          debugTrace(`Failed to fetch ${scanType} scan status: ${response.status} ${response.statusText}`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const data = await response.json();
        const scans = Array.isArray(data) ? data : (data.scans || []);
        debugTrace(`Retrieved ${scans.length} ${scanType} scans`);
        if (!scans || !Array.isArray(scans) || scans.length === 0) {
          debugTrace(`No ${scanType} scans found, will check again`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const mostRecentScan = scans.reduce((latest, scan) => {
          const scanDate = new Date(scan.created_at);
          return scanDate > new Date(latest.created_at) ? scan : latest;
        }, scans[0]);
        debugTrace(`Most recent ${scanType} scan status: ${mostRecentScan.status}, ID: ${mostRecentScan.id || 'unknown'}`);
        setMostRecentScanStatus(mostRecentScan.status);
        if (setMostRecentScan) {
          setMostRecentScan(mostRecentScan);
          debugTrace(`Updated UI with most recent ${scanType} scan data`);
        }
        if (mostRecentScan.status === 'completed' || 
            mostRecentScan.status === 'success' || 
            mostRecentScan.status === 'failed' || 
            mostRecentScan.status === 'error') {
          debugTrace(`${scanType} scan finished with status: ${mostRecentScan.status}`);
          setIsScanning(false);
          resolve(mostRecentScan);
        } else if (mostRecentScan.status === 'processing') {
          debugTrace(`${scanType} scan is still processing large results, checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        } else {
          debugTrace(`${scanType} scan still pending (status: ${mostRecentScan.status}), checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        }
      } catch (error) {
        debugTrace(`Error checking ${scanType} scan status: ${error.message}\n${error.stack}`);
        setTimeout(checkStatus, 5000);
      }
    };
    checkStatus();
  });
};

// Custom function to wait for both CeWL and ShuffleDNS Custom scans
const waitForCeWLAndShuffleDNSCustom = async (
  activeTarget,
  setIsCeWLScanning,
  setMostRecentCeWLScanStatus,
  setMostRecentCeWLScan,
  setMostRecentShuffleDNSCustomScanStatus,
  setMostRecentShuffleDNSCustomScan
) => {
  debugTrace(`Starting waitForCeWLAndShuffleDNSCustom for target ${activeTarget.id}`);
  await waitForScanCompletion(
    'cewl',
    activeTarget.id,
    setIsCeWLScanning,
    setMostRecentCeWLScanStatus,
    setMostRecentCeWLScan
  );
  debugTrace(`CeWL scan completed, now waiting for ShuffleDNS custom scan`);
  return new Promise((resolve) => {
    let attempts = 0;
    const checkStatus = async () => {
      attempts++;
      debugTrace(`Checking ShuffleDNS custom scan status - attempt #${attempts}`);
      try {
        const url = `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/scope-targets/${activeTarget.id}/shufflednscustom-scans`;
        debugTrace(`Fetching ShuffleDNS custom scan status from: ${url}`);
        const response = await fetch(url);
        if (!response.ok) {
          debugTrace(`Failed to fetch ShuffleDNS custom scan status: ${response.status} ${response.statusText}`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const scans = await response.json();
        debugTrace(`Retrieved ${scans?.length || 0} ShuffleDNS custom scans`);
        if (!scans || !Array.isArray(scans) || scans.length === 0) {
          debugTrace(`No ShuffleDNS custom scans found, will check again`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const mostRecentScan = scans[0];
        debugTrace(`Most recent ShuffleDNS custom scan status: ${mostRecentScan.status}, ID: ${mostRecentScan.id || 'unknown'}`);
        setMostRecentShuffleDNSCustomScanStatus(mostRecentScan.status);
        setMostRecentShuffleDNSCustomScan(mostRecentScan);
        if (mostRecentScan.status === 'completed' || 
            mostRecentScan.status === 'success' || 
            mostRecentScan.status === 'failed' || 
            mostRecentScan.status === 'error') {
          debugTrace(`ShuffleDNS custom scan finished with status: ${mostRecentScan.status}`);
          resolve(mostRecentScan);
        } else if (mostRecentScan.status === 'processing') {
          debugTrace(`ShuffleDNS custom scan is still processing large results, checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        } else {
          debugTrace(`ShuffleDNS custom scan still pending (status: ${mostRecentScan.status}), checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        }
      } catch (error) {
        debugTrace(`Error checking ShuffleDNS custom scan status: ${error.message}\n${error.stack}`);
        setTimeout(checkStatus, 5000);
      }
    };
    checkStatus();
  });
};

// Function to resume auto scan from a specific step
const resumeAutoScan = async (
  fromStep,
  activeTarget,
  getAutoScanSteps,
  setIsAutoScanning,
  setAutoScanCurrentStep
) => {
  try {
    setIsAutoScanning(false);
    let startFromIndex = 0;
    const steps = getAutoScanSteps(activeTarget);
    for (let i = 0; i < steps.length; i++) {
      if (steps[i].name === fromStep) {
        startFromIndex = i;
        break;
      }
    }
    
    // Execute steps from the determined starting point
    for (let i = startFromIndex; i < steps.length; i++) {
      try {
        await steps[i].action();
      } catch (error) {
        debugTrace(`Error in step ${i+1}/${steps.length}: ${error.message}`);
      }
      
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
  } catch (error) {
    debugTrace(`Error resuming Auto Scan: ${error.message}`);
  } finally {
    setIsAutoScanning(false);
    setAutoScanCurrentStep(AUTO_SCAN_STEPS.COMPLETED);
    
    // Update the API with completed status
    await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED);
  }
};

// Function to start a new auto scan
const startAutoScan = async (
  activeTarget,
  setIsAutoScanning,
  setAutoScanCurrentStep,
  setAutoScanTargetId,
  getAutoScanSteps,
  consolidatedSubdomains,
  mostRecentHttpxScan,
  autoScanSessionId
) => {
  if (!activeTarget || !activeTarget.id) {
    console.log("No active target selected.");
    return;
  }

  setIsAutoScanning(true);
  setAutoScanCurrentStep(AUTO_SCAN_STEPS.IDLE);
  setAutoScanTargetId(activeTarget.id);
  
  await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.IDLE);
  
  try {
    const steps = getAutoScanSteps();
    for (let i = 0; i < steps.length; i++) {
      try {
        await steps[i].action();
      } catch (error) {
        debugTrace(`Error in step ${steps[i].name}: ${error.message}`);
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    debugTrace("All auto scan steps completed");
  } catch (error) {
    debugTrace(`ERROR during Auto Scan: ${error.message}`);
  } finally {
    debugTrace("Auto Scan session finalizing - setting state to COMPLETED");
    setIsAutoScanning(false);
    setAutoScanCurrentStep(AUTO_SCAN_STEPS.COMPLETED);
    await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED);
    debugTrace("Auto Scan session ended");
    if (autoScanSessionId) {
      let finalConsolidatedSubdomains = Array.isArray(consolidatedSubdomains) ? consolidatedSubdomains.length : 0;
      let finalLiveWebServers = 0;
      if (mostRecentHttpxScan && mostRecentHttpxScan.result && typeof mostRecentHttpxScan.result.String === 'string') {
        finalLiveWebServers = mostRecentHttpxScan.result.String.split('\n').filter(line => line.trim()).length;
      }
      try {
        await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan/session/${autoScanSessionId}/final-stats`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              final_consolidated_subdomains: finalConsolidatedSubdomains,
              final_live_web_servers: finalLiveWebServers
            })
          }
        );
      } catch (err) {
        debugTrace('Failed to update final stats for auto scan session: ' + err.message);
      }
    }
  }
};

// Helper to check and resume auto scan
const checkAndResumeAutoScan = (
  storedStep,
  storedTargetId,
  scopeTargets,
  activeTarget,
  setIsAutoScanning,
  setAutoScanCurrentStep,
  setAutoScanTargetId,
  resumeAutoScanFn
) => {
  if (storedStep && storedStep !== AUTO_SCAN_STEPS.IDLE && storedStep !== AUTO_SCAN_STEPS.COMPLETED && storedTargetId) {
    console.log(`Detected in-progress Auto Scan (step: ${storedStep}). Attempting to resume...`);
    
    // Find the target with the matching ID
    const matchingTarget = scopeTargets.find(target => target.id === storedTargetId);
    
    if (matchingTarget && matchingTarget.id === activeTarget?.id) {
      setIsAutoScanning(true);
      resumeAutoScanFn(storedStep);
    } else {
      // Reset the auto scan state on the server
      updateAutoScanState(storedTargetId, AUTO_SCAN_STEPS.IDLE);
      
      setAutoScanCurrentStep(AUTO_SCAN_STEPS.IDLE);
      setAutoScanTargetId(null);
    }
  }
};

export {
  waitForScanCompletion,
  AUTO_SCAN_STEPS,
  debugTrace,
  resumeAutoScan,
  startAutoScan,
  checkAndResumeAutoScan,
  waitForCeWLAndShuffleDNSCustom,
  updateAutoScanState
}; 
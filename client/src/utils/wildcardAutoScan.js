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
  GOSPIDER: 'gospider', // 11
  SUBDOMAINIZER: 'subdomainizer', // 12
  NUCLEI_SCREENSHOT: 'nuclei-screenshot', // 13
  METADATA: 'metadata', // 14
  COMPLETED: 'completed' // 15
};

// Debug utility function
const debugTrace = (message) => {
  const timestamp = new Date().toISOString();
  console.log(`[TRACE ${timestamp}] ${message}`);
};

// Helper function to wait for a scan to complete
const waitForScanCompletion = async (scanType, targetId, setIsScanning, setMostRecentScanStatus, setMostRecentScan = null) => {
  debugTrace(`waitForScanCompletion started for ${scanType}`);
  
  // Add a hard safety timeout in case the promise never resolves
  return Promise.race([
    new Promise((resolve) => {
      const startTime = Date.now();
      const maxWaitTime = 10 * 60 * 1000; // 10 minutes maximum wait
      const hardMaxWaitTime = 15 * 60 * 1000; // 15 minutes absolute maximum
      let attempts = 0;
      
      // Add a hard timeout as safety
      const hardTimeout = setTimeout(() => {
        debugTrace(`HARD TIMEOUT: ${scanType} scan exceeded maximum wait time of 15 minutes`);
        setIsScanning(false);
        resolve({ status: 'hard_timeout', message: 'Hard scan timeout exceeded' });
      }, hardMaxWaitTime);
      
      const checkStatus = async () => {
        attempts++;
        debugTrace(`Checking ${scanType} scan status - attempt #${attempts}`);
        try {
          // Check if we've been waiting too long
          if (Date.now() - startTime > maxWaitTime) {
            debugTrace(`${scanType} scan taking too long (${Math.round((Date.now() - startTime)/1000)}s), proceeding with next step`);
            setIsScanning(false);
            clearTimeout(hardTimeout); // Clear the hard timeout
            return resolve({ status: 'timeout', message: 'Scan timeout exceeded' });
          }
          
          const url = `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${targetId}/scans/${scanType}`;
          debugTrace(`Fetching scan status from: ${url}`);
          
          const response = await fetch(url);
          
          if (!response.ok) {
            debugTrace(`Failed to fetch ${scanType} scan status: ${response.status} ${response.statusText}`);
            
            // If we get a 404 or other error after multiple attempts, let's proceed rather than getting stuck
            if (attempts > 10) {
              debugTrace(`${scanType} scan failed to fetch status after ${attempts} attempts, proceeding with next step`);
              setIsScanning(false);
              clearTimeout(hardTimeout); // Clear the hard timeout
              return resolve({ status: 'error', message: 'Failed to fetch scan status' });
            }
            
            // If we get a 404 or other error, we'll check again after a delay
            setTimeout(checkStatus, 5000);
            return;
          }
          
          const scans = await response.json();
          debugTrace(`Retrieved ${scans?.length || 0} ${scanType} scans`);
          
          // Handle case where there are no scans after multiple attempts
          if (!scans || !Array.isArray(scans) || scans.length === 0) {
            debugTrace(`No ${scanType} scans found, will check again`);
            
            if (attempts > 10) {
              debugTrace(`${scanType} scan returned no scans after ${attempts} attempts, proceeding with next step`);
              setIsScanning(false);
              clearTimeout(hardTimeout); // Clear the hard timeout
              return resolve({ status: 'no_scans', message: 'No scans found' });
            }
            
            setTimeout(checkStatus, 5000);
            return;
          }
          
          // Find the most recent scan
          const mostRecentScan = scans.reduce((latest, scan) => {
            const scanDate = new Date(scan.created_at);
            return scanDate > new Date(latest.created_at) ? scan : latest;
          }, scans[0]);
          
          debugTrace(`Most recent ${scanType} scan status: ${mostRecentScan.status}, ID: ${mostRecentScan.id || 'unknown'}`);
          
          // Update status in UI
          setMostRecentScanStatus(mostRecentScan.status);
          
          // Also update the most recent scan object if setter is provided
          if (setMostRecentScan) {
            setMostRecentScan(mostRecentScan);
            debugTrace(`Updated UI with most recent ${scanType} scan data`);
          }
          
          if (mostRecentScan.status === 'completed' || 
              mostRecentScan.status === 'success' || 
              mostRecentScan.status === 'failed' || 
              mostRecentScan.status === 'error') {  // Also consider 'error' status as completed
            debugTrace(`${scanType} scan finished with status: ${mostRecentScan.status}`);
            setIsScanning(false);
            clearTimeout(hardTimeout); // Clear the hard timeout
            resolve(mostRecentScan);
          } else if (mostRecentScan.status === 'processing') {
            // The scan is complete but still processing large results (e.g., GAU with >1000 URLs)
            debugTrace(`${scanType} scan is still processing large results, checking again in 5 seconds`);
            setTimeout(checkStatus, 5000);
          } else {
            // Still pending or another status, check again after delay
            debugTrace(`${scanType} scan still pending (status: ${mostRecentScan.status}), checking again in 5 seconds`);
            setTimeout(checkStatus, 5000);
          }
        } catch (error) {
          debugTrace(`Error checking ${scanType} scan status: ${error.message}\n${error.stack}`);
          
          // If we have persistent errors after multiple attempts, proceed rather than getting stuck
          if (attempts > 10) {
            debugTrace(`${scanType} scan had persistent errors after ${attempts} attempts, proceeding with next step`);
            setIsScanning(false);
            clearTimeout(hardTimeout); // Clear the hard timeout
            return resolve({ status: 'persistent_error', message: 'Persistent errors checking scan status' });
          }
          
          // Don't reject immediately on errors, try again after a delay
          setTimeout(checkStatus, 5000);
        }
      };
      
      // Start checking status immediately
      checkStatus();
    }),
    // Add a separate timeout promise as a backstop
    new Promise((resolve) => {
      setTimeout(() => {
        debugTrace(`BACKUP TIMEOUT: ${scanType} scan timed out at 20 minutes absolute maximum`);
        setIsScanning(false);
        resolve({ status: 'absolute_timeout', message: 'Absolute timeout exceeded' });
      }, 20 * 60 * 1000); // 20 minutes absolute maximum
    })
  ]);
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
    localStorage.setItem('autoScanCurrentStep', AUTO_SCAN_STEPS.COMPLETED);
  }
};

// Function to start a new auto scan
const startAutoScan = async (
  activeTarget,
  setIsAutoScanning,
  setAutoScanCurrentStep,
  setAutoScanTargetId,
  getAutoScanSteps
) => {
  if (!activeTarget || !activeTarget.id) {
    console.log("No active target selected.");
    return;
  }

  setIsAutoScanning(true);
  setAutoScanCurrentStep(AUTO_SCAN_STEPS.IDLE);
  setAutoScanTargetId(activeTarget.id);
  
  localStorage.setItem('autoScanTargetId', activeTarget.id);
  localStorage.setItem('autoScanCurrentStep', AUTO_SCAN_STEPS.IDLE);
  debugTrace("localStorage initialized: autoScanTargetId=" + activeTarget.id + ", autoScanCurrentStep=" + AUTO_SCAN_STEPS.IDLE);
  
  try {
    const steps = getAutoScanSteps(activeTarget);
    
    // Execute all scan steps in sequence
    for (let i = 1; i < steps.length; i++) {
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
    // Only clear the localStorage at the very end when we're completely done
    debugTrace("Auto Scan session finalizing - setting state to COMPLETED");
    setIsAutoScanning(false);
    setAutoScanCurrentStep(AUTO_SCAN_STEPS.COMPLETED);
    localStorage.setItem('autoScanCurrentStep', AUTO_SCAN_STEPS.COMPLETED);
    debugTrace("localStorage updated: autoScanCurrentStep=" + AUTO_SCAN_STEPS.COMPLETED);
    debugTrace("Auto Scan session ended");
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
      localStorage.removeItem('autoScanCurrentStep');
      localStorage.removeItem('autoScanTargetId');
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
  checkAndResumeAutoScan
}; 
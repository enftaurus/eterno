import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { Shield, FileText, Lock, ExternalLink, Heart } from 'lucide-react';

export const AppFooter: React.FC = () => {
  const [showPrivacy, setShowPrivacy] = useState(false);
  const [showTerms, setShowTerms] = useState(false);
  const [showSecurity, setShowSecurity] = useState(false);

  return (
    <>
      <footer className="mt-auto border-t border-neutral-200/80 bg-white/60 py-6 px-6 text-xs text-neutral-500">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-neutral-800">Code Crusaders</span>
            <span className="text-neutral-300">•</span>
            <span>Developer Intelligence Platform by Eterno</span>
            <span className="text-neutral-300">•</span>
            <span>© {new Date().getFullYear()}</span>
          </div>

          <div className="flex flex-wrap items-center gap-5">
            <button
              onClick={() => setShowPrivacy(true)}
              className="hover:text-neutral-900 transition-colors flex items-center gap-1"
            >
              <Lock className="w-3.5 h-3.5 text-neutral-400" />
              Privacy Policy
            </button>
            <button
              onClick={() => setShowTerms(true)}
              className="hover:text-neutral-900 transition-colors flex items-center gap-1"
            >
              <FileText className="w-3.5 h-3.5 text-neutral-400" />
              Terms of Service
            </button>
            <button
              onClick={() => setShowSecurity(true)}
              className="hover:text-neutral-900 transition-colors flex items-center gap-1"
            >
              <Shield className="w-3.5 h-3.5 text-neutral-400" />
              Security & Verification
            </button>
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Zero-Mock Operational
            </span>
          </div>
        </div>
      </footer>

      {/* Privacy Policy Modal */}
      <Modal
        isOpen={showPrivacy}
        onClose={() => setShowPrivacy(false)}
        title="Privacy Policy & Data Ethics"
        description="Last updated: October 2026. How Code Crusaders collects, processes, and protects your developer data."
      >
        <div className="space-y-4 text-xs text-neutral-600 max-h-96 overflow-y-auto pr-1">
          <section>
            <h4 className="font-bold text-neutral-900 mb-1">1. Information We Collect</h4>
            <p className="leading-relaxed">
              Code Crusaders only accesses public developer data through verified external platform interfaces (LeetCode, GitHub, and Codeforces).
              We collect:
            </p>
            <ul className="list-disc pl-5 mt-1 space-y-0.5 text-neutral-500">
              <li>Public submission verdicts, timestamps, and problem slugs</li>
              <li>Public repository commits and contribution activity</li>
              <li>Competitive programming contest ratings and rank statistics</li>
              <li>Institutional roster identifiers provided by your college</li>
            </ul>
          </section>

          <section>
            <h4 className="font-bold text-neutral-900 mb-1">2. Zero Credential Storage</h4>
            <p className="leading-relaxed">
              We <strong>never</strong> ask for or store your third-party platform passwords. Account verification is performed cryptographically:
              via exact-token bio matching on LeetCode, public API handle checks on Codeforces, or standard OAuth 2.0 authorization on GitHub.
            </p>
          </section>

          <section>
            <h4 className="font-bold text-neutral-900 mb-1">3. Transparent Developer Score</h4>
            <p className="leading-relaxed">
              Your 0–100 Developer Score is computed entirely from deterministic, public mathematical formulas based on consistency, problem-solving, competitive performance, and engineering activity. There are no proprietary black-box algorithms or opaque ranking penalties.
            </p>
          </section>

          <section>
            <h4 className="font-bold text-neutral-900 mb-1">4. Data Retention & Tenant Isolation</h4>
            <p className="leading-relaxed">
              Your data is strictly partitioned by institutional tenancy. Colleges cannot access other institutions' private challenge answers or student rosters. You may request account unlinking or deletion at any time.
            </p>
          </section>
        </div>
      </Modal>

      {/* Terms of Service Modal */}
      <Modal
        isOpen={showTerms}
        onClose={() => setShowTerms(false)}
        title="Terms of Service"
        description="Terms governing the use of the Code Crusaders platform and college coding challenges."
      >
        <div className="space-y-4 text-xs text-neutral-600 max-h-96 overflow-y-auto pr-1">
          <section>
            <h4 className="font-bold text-neutral-900 mb-1">1. Academic Integrity & Honest Solutions</h4>
            <p className="leading-relaxed">
              Submissions verified for college coding challenges must represent your original work. Verifications inspect exact submission timestamps to ensure problems were solved after challenge assignment baselines and before published deadlines.
            </p>
          </section>

          <section>
            <h4 className="font-bold text-neutral-900 mb-1">2. Multi-Account & Impersonation Prohibition</h4>
            <p className="leading-relaxed">
              Each student is entitled to link exactly one personal profile per external coding platform. Account reassignment or claiming third-party profiles without cryptographic token proof is strictly prohibited and audited.
            </p>
          </section>

          <section>
            <h4 className="font-bold text-neutral-900 mb-1">3. Service Availability</h4>
            <p className="leading-relaxed">
              Code Crusaders synchronizes external platform records asynchronously. We respect third-party rate limits and implement exponential backoff during upstream maintenance windows.
            </p>
          </section>
        </div>
      </Modal>

      {/* Security & Verification Modal */}
      <Modal
        isOpen={showSecurity}
        onClose={() => setShowSecurity(false)}
        title="Security & Platform Architecture"
        description="Our technical safeguards and cryptographic verification standards."
      >
        <div className="space-y-3 text-xs text-neutral-600 max-h-96 overflow-y-auto pr-1">
          <p className="leading-relaxed">
            Code Crusaders enforces enterprise-grade security protocols:
          </p>
          <div className="p-3 bg-neutral-50 rounded-lg border border-neutral-200/80 space-y-2">
            <div>
              <span className="font-semibold text-neutral-900 block">Strict Domain Boundary Validation</span>
              <span className="text-[11px] text-neutral-500">Problem URLs must strictly match verified provider domains; spoofed URLs and userinfo injection are rejected.</span>
            </div>
            <div>
              <span className="font-semibold text-neutral-900 block">Formula Injection Neutralization</span>
              <span className="text-[11px] text-neutral-500">All student CSV exports and roster imports neutralize spreadsheet executable tokens (=, +, -, @).</span>
            </div>
            <div>
              <span className="font-semibold text-neutral-900 block">Timestamp Verification Invariants</span>
              <span className="text-[11px] text-neutral-500">Clock-skew protection disallows future-dated evidence, while challenge baselines guarantee real-time solve honesty.</span>
            </div>
          </div>
        </div>
      </Modal>
    </>
  );
};
